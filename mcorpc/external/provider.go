package external

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-config"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
	"github.com/sirupsen/logrus"
)

// agents we do not ever wish to load from external agents
var denylist = []string{"rpcutil", "choria_util", "discovery"}

// Provider is a Choria Agent Provider that supports calling agents external to the
// choria process written in any language
type Provider struct {
	cfg    *config.Config
	log    *logrus.Entry
	agents []*agent.DDL
	paths  map[string]string
}

// Initialize configures the agent provider
func (p *Provider) Initialize(fw *choria.Framework, log *logrus.Entry) {
	p.cfg = fw.Configuration()
	p.log = log.WithFields(logrus.Fields{"provider": "external"})
	p.paths = map[string]string{}

	p.loadAgents()
}

// RegisterAgents registers known ruby agents using a shimm agent
func (p *Provider) RegisterAgents(ctx context.Context, mgr server.AgentManager, connector choria.InstanceConnector, log *logrus.Entry) error {
	for _, ddl := range p.Agents() {
		agent, err := p.newExternalAgent(ddl, mgr)
		if err != nil {
			p.log.Errorf("Could not register external agent %s: %s", agent.Name(), err)
			continue
		}

		err = mgr.RegisterAgent(ctx, agent.Name(), agent, connector)
		if err != nil {
			p.log.Errorf("Could not register external agent %s: %s", agent.Name(), err)
			continue
		}
	}

	return nil
}

// Agents provides a list of loaded agent DDLs
func (p *Provider) Agents() []*agent.DDL {
	return p.agents
}

// Version reports the version for this provider
func (p *Provider) Version() string {
	return fmt.Sprintf("%s version %s", p.PluginName(), p.PluginVersion())
}

func (p *Provider) loadAgents() {
	p.eachAgent(func(a *agent.DDL) {
		p.log.Debugf("Found external DDL for agent %s", a.Metadata.Name)
		p.agents = append(p.agents, a)
		p.paths[a.Metadata.Name] = a.SourceLocation
	})
}

func (p *Provider) agentDDL(a string) (*agent.DDL, bool) {
	for _, agent := range p.agents {
		if agent.Metadata.Name == a {
			return agent, true
		}
	}

	return nil, false
}

func (p *Provider) eachAgent(cb func(ddl *agent.DDL)) {
	for _, dir := range p.cfg.Choria.RubyLibdir {
		agentsdir := filepath.Join(dir, "mcollective", "agent")

		p.log.Debugf("Attempting to load External agents from %s", agentsdir)

		err := filepath.Walk(agentsdir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			fname := info.Name()
			ext := filepath.Ext(fname)
			name := strings.TrimSuffix(fname, ext)

			if ext != ".json" {
				return nil
			}

			if !shouldLoadAgent(name) {
				p.log.Warnf("External agents are not allowed to supply an agent called '%s', skipping", name)
				return nil
			}

			p.log.Debugf("Attempting to load %s as an agent DDL", path)

			ddl, err := agent.New(path)
			if err != nil {
				p.log.Errorf("Could not load external agent DDL %s: %s", path, err)
				return nil
			}

			if ddl.Metadata.Provider == "external" {
				cb(ddl)
			}

			return nil
		})

		if err != nil {
			p.log.Errorf("Could not find agents in %s: %s", agentsdir, err)
		}

	}
}

func shouldLoadAgent(name string) bool {
	for _, a := range denylist {
		if a == name {
			return false
		}
	}

	return true
}
