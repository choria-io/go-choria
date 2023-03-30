// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/server"
)

var (
	// agents we do not ever wish to load from external agents
	denyList = []string{"rpcutil", "choria_util", "choria_provision", "choria_registry", "discovery", "scout"}
	// how frequently agents are reconciled
	watchInterval = time.Minute
	// we only consider ddl files modified longer than this ago for reconciliation
	fileChangeGrace = 20 * time.Second
)

// Provider is a Choria Agent Provider that supports calling agents external to the
// choria process written in any language
type Provider struct {
	cfg    *config.Config
	log    *logrus.Entry
	agents []*agent.DDL
	paths  map[string]string
	mu     sync.Mutex
}

// Initialize configures the agent provider
func (p *Provider) Initialize(cfg *config.Config, log *logrus.Entry) {
	p.cfg = cfg
	p.log = log.WithFields(logrus.Fields{"provider": "external"})
	p.paths = map[string]string{}
}

// RegisterAgents registers known ruby agents using a shim agent and starts a background reconciliation loop to add/remove/update agents without restarts
func (p *Provider) RegisterAgents(ctx context.Context, mgr server.AgentManager, connector inter.AgentConnector, log *logrus.Entry) error {
	go p.watchAgents(ctx, mgr, connector)

	return nil
}

func (p *Provider) upgradeExistingAgents(foundAgents []*agent.DDL, mgr server.AgentManager) error {
	for i, currentDDL := range p.agents {
		candidateDDL := findInAgentList(foundAgents, func(a *agent.DDL) bool {
			if a.Metadata.Name != currentDDL.Metadata.Name {
				return false
			}

			// we check the ddl location so that moving a agent to a different place, even when versions match will also reload it
			if a.Metadata.Version == currentDDL.Metadata.Version && a.SourceLocation == currentDDL.SourceLocation {
				return false
			}

			return p.shouldProcessModifiedDDL(a.SourceLocation)
		})

		if candidateDDL == nil {
			continue
		}

		newAgent, err := p.newExternalAgent(candidateDDL, mgr)
		if err != nil {
			p.log.Errorf("Could not create upgraded external agent %v: %v", candidateDDL.Metadata.Name, err)
			continue
		}

		err = mgr.ReplaceAgent(candidateDDL.Metadata.Name, newAgent)
		if err != nil {
			p.log.Errorf("Could not replace upgraded external agent %v: %v", candidateDDL.Metadata.Name, err)
			continue
		}

		p.agents[i] = candidateDDL
		p.paths[candidateDDL.Metadata.Name] = candidateDDL.SourceLocation
	}

	return nil
}

func (p *Provider) removeOrphanAgents(foundAgents []*agent.DDL, mgr server.AgentManager, connector inter.AgentConnector) error {
	var remove []int

	for i, known := range p.agents {
		found := findInAgentList(foundAgents, func(a *agent.DDL) bool {
			return a.Metadata.Name == known.Metadata.Name
		})

		if found == nil {
			p.log.Infof("Removing agent %s after the DDL %s was removed", known.Metadata.Name, known.SourceLocation)
			err := mgr.UnregisterAgent(known.Metadata.Name, connector)
			if err != nil {
				p.log.Errorf("Could not unregister agent %v: %v", known.Metadata.Name, err)
				continue
			}

			delete(p.paths, known.Metadata.Name)
			remove = append(remove, i)
		}
	}

	for _, i := range remove {
		p.agents = append(p.agents[:i], p.agents[i+1:]...)
	}

	return nil
}

func (p *Provider) registerNewAgents(ctx context.Context, foundAgents []*agent.DDL, mgr server.AgentManager, connector inter.AgentConnector) error {
	for _, candidateDDL := range foundAgents {
		found := findInAgentList(p.agents, func(a *agent.DDL) bool {
			return candidateDDL.Metadata.Name == a.Metadata.Name
		})

		if found == nil && p.shouldProcessModifiedDDL(candidateDDL.SourceLocation) {
			p.log.Debugf("Registering new agent %v version %v from %s", candidateDDL.Metadata.Name, candidateDDL.Metadata.Version, candidateDDL.SourceLocation)
			agent, err := p.newExternalAgent(candidateDDL, mgr)
			if err != nil {
				p.log.Errorf("Could not register external agent %s: %s", agent.Name(), err)
				continue
			}

			err = mgr.RegisterAgent(ctx, agent.Name(), agent, connector)
			if err != nil {
				p.log.Errorf("Could not register external agent %s: %s", agent.Name(), err)
				continue
			}

			p.agents = append(p.agents, candidateDDL)
			p.paths[candidateDDL.Metadata.Name] = candidateDDL.SourceLocation
		}
	}

	return nil
}

func (p *Provider) shouldProcessModifiedDDL(path string) bool {
	if path == "" {
		return false
	}

	stat, err := os.Stat(path)
	if err != nil {
		p.log.Errorf("Could not determine age of DDL file %v: %v", path, err)
		return false
	}

	since := time.Since(stat.ModTime())
	if since < fileChangeGrace {
		p.log.Debugf("Skipping updated DDL file %v that is %v old", path, since)
		return false
	}

	return true
}

func (p *Provider) reconcileAgents(ctx context.Context, mgr server.AgentManager, connector inter.AgentConnector) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.log.Debugf("Reconciling external agents from disk with running agents")

	var foundAgents []*agent.DDL
	p.eachAgent(func(candidateDDL *agent.DDL) {
		if candidateDDL.SourceLocation == "" {
			return
		}

		foundAgents = append(foundAgents, candidateDDL)
	})

	p.log.Debugf("Found %d external agents on disk", len(foundAgents))

	err := p.registerNewAgents(ctx, foundAgents, mgr, connector)
	if err != nil {
		p.log.Warnf("Could not register new agents: %v", err)
	}

	err = p.upgradeExistingAgents(foundAgents, mgr)
	if err != nil {
		p.log.Warnf("Could not upgrade existing agents: %v", err)
	}

	err = p.removeOrphanAgents(foundAgents, mgr, connector)
	if err != nil {
		p.log.Warnf("Could not remove orphaned agents: %v", err)
	}

	return nil
}

func (p *Provider) watchAgents(ctx context.Context, mgr server.AgentManager, connector inter.AgentConnector) {
	err := p.reconcileAgents(ctx, mgr, connector)
	if err != nil {
		p.log.Errorf("Initial agent reconcile failed: %v", err)
	}

	ticker := time.NewTicker(watchInterval)
	p.log.Debugf("Watching for agent updates every %v", watchInterval)

	for {
		select {
		case <-ticker.C:
			err := p.reconcileAgents(ctx, mgr, connector)
			if err != nil {
				p.log.Errorf("Reconciling agents failed: %v", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

// Agents provides a list of loaded agent DDLs
func (p *Provider) Agents() []*agent.DDL {
	p.mu.Lock()
	defer p.mu.Unlock()

	dst := make([]*agent.DDL, len(p.agents))
	copy(dst, p.agents)

	return dst
}

// Version reports the version for this provider
func (p *Provider) Version() string {
	return fmt.Sprintf("%s version %s", p.PluginName(), p.PluginVersion())
}

func (p *Provider) agentDDL(a string) (*agent.DDL, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, agent := range p.agents {
		if agent.Metadata.Name == a {
			return agent, true
		}
	}

	return nil, false
}

// walks the plugin.choria.agent_provider.mcorpc.libdir directories looking for agents.
//
// we support $dir/agent.json and $dir/agent/agent.json
func (p *Provider) eachAgent(cb func(ddl *agent.DDL)) {
	for _, libDir := range p.cfg.Choria.RubyLibdir {
		agentsDir := filepath.Join(libDir, "mcollective", "agent")

		p.log.Debugf("Attempting to load External agents from %s", agentsDir)

		err := filepath.WalkDir(agentsDir, func(path string, info fs.DirEntry, err error) error {
			if err != nil || path == agentsDir {
				return err
			}

			// if early on we decide to skip dir, this will hold that and used everywhere we return on error
			var retErr error

			// either x.json or x in the case of a directory holding a ddl
			fname := info.Name()

			// full path, which in the case of a directory holding a ddl will be adjusted to the nested one
			ddlPath := path

			if info.IsDir() {
				// We dont want to keep walking into directory so we check if the
				// ddl matching fname exist then just use that, but we avoid
				// traversing nested directories
				ddlPath = filepath.Join(path, fmt.Sprintf("%s.json", fname))
				retErr = fs.SkipDir
			}

			if !util.FileExist(ddlPath) {
				return retErr
			}

			ext := filepath.Ext(ddlPath)
			name := strings.TrimSuffix(fname, ext)

			if ext != ".json" {
				return retErr
			}

			p.log.Debugf("Attempting to load %s as an agent DDL", ddlPath)
			ddl, err := agent.New(ddlPath)
			if err != nil {
				p.log.Errorf("Could not load agent DDL %s: %s", ddlPath, err)
				return retErr
			}

			if ddl.Metadata.Provider != "external" {
				return nil
			}

			if !shouldLoadAgent(name) {
				p.log.Warnf("External agents are not allowed to supply an agent called '%s', skipping", name)
				return retErr
			}

			cb(ddl)

			return retErr
		})

		if err != nil {
			p.log.Errorf("Could not find agents in %s: %s", agentsDir, err)
		}
	}
}

func findInAgentList(agents []*agent.DDL, cb func(*agent.DDL) bool) *agent.DDL {
	for _, d := range agents {
		if cb(d) {
			return d
		}
	}

	return nil
}

func shouldLoadAgent(name string) bool {
	for _, a := range denyList {
		if a == name {
			return false
		}
	}

	return true
}
