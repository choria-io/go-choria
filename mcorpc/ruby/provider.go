package ruby

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-config"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
	"github.com/sirupsen/logrus"
)

// agents we do not ever wish to load from ruby
var denylist = []string{"rpcutil", "choria_util", "discovery"}

// Provider is a Agent Provider capable of executing old mcollective ruby agents
type Provider struct {
	fw     *choria.Framework
	cfg    *config.Config
	log    *logrus.Entry
	agents []*agent.DDL
}

// New creates a new provider that will find ruby agents in the configured libdirs
func New(fw *choria.Framework) *Provider {
	p := &Provider{}
	p.Initialize(fw, logrus.WithFields(logrus.Fields{"provider": "ruby"}))

	return p
}

// Initialize configures the agent provider
func (p *Provider) Initialize(fw *choria.Framework, log *logrus.Entry) {
	p.fw = fw
	p.cfg = fw.Config
	p.log = log.WithFields(logrus.Fields{"provider": "ruby"})

	p.loadAgents(fw.Config.Choria.RubyLibdir)
}

// RegisterAgents registers known ruby agents using a shimm agent
func (p *Provider) RegisterAgents(ctx context.Context, mgr server.AgentManager, connector choria.InstanceConnector, log *logrus.Entry) error {
	for _, ddl := range p.Agents() {
		agent, err := NewRubyAgent(ddl, mgr)
		if err != nil {
			p.log.Errorf("Could not register Ruby agent %s: %s", ddl.Metadata.Name, err)
			continue
		}

		err = mgr.RegisterAgent(ctx, agent.Name(), agent, connector)
		if err != nil {
			p.log.Errorf("Could not register Ruby agent %s: %s", ddl.Metadata.Name, err)
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
