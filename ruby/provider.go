package ruby

import (
	"context"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

type AgentManager interface {
	RegisterAgent(ctx context.Context, name string, agent agents.Agent, conn choria.AgentConnector) error
	Logger() *logrus.Entry
	Choria() *choria.Framework
}

// agents we do not ever wish to load from ruby
var denylist = []string{"rpcutil", "choria_util", "discovery"}

// Provider is a Agent Provider capable of executing old mcollective ruby agents
type Provider struct {
	fw     *choria.Framework
	cfg    *choria.Config
	log    *logrus.Entry
	agents []*agent.DDL
}

// New creates a new provider that will find ruby agents in the configured libdirs
func New(fw *choria.Framework) *Provider {
	p := &Provider{
		fw:  fw,
		cfg: fw.Config,
		log: logrus.WithFields(logrus.Fields{"provider": "ruby"}),
	}

	p.loadAgents(fw.Config.LibDir)

	return p
}

// RegisterAgents registers known ruby agents using a shimm agent
func (p *Provider) RegisterAgents(ctx context.Context, mgr AgentManager, connector choria.InstanceConnector, log *logrus.Entry) error {
	for _, ddl := range p.Agents() {
		agent, err := NewRubyAgent(ddl, mgr)
		if err != nil {
			p.log.Errorf("Could not register Ruby agent %s: %s", ddl.Metadata.Name, err)
			continue
		}

		mgr.RegisterAgent(ctx, agent.Name(), agent, connector)
	}

	return nil
}

// Agents provides a list of loaded agent DDLs
func (p *Provider) Agents() []*agent.DDL {
	return p.agents
}
