package server

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/sirupsen/logrus"
)

// AgentProvider is capable of adding agents into a running instance
type AgentProvider interface {
	Initialize(fw *choria.Framework, log *logrus.Entry)
	RegisterAgents(ctx context.Context, mgr AgentManager, connector choria.InstanceConnector, log *logrus.Entry) error
	Version() string
}

var additionalAgentProviders []AgentProvider
var aapmu *sync.Mutex

func init() {
	additionalAgentProviders = []AgentProvider{}
	aapmu = &sync.Mutex{}
}

// RegisterAdditionalAgentProvider registers an agent provider as a subsystem
// capable of delivering new types of agent like the legacy mcollective ruby compatible
// ones
//
// Custom builders can use this to extend choria with new agent capabilities
func RegisterAdditionalAgentProvider(p AgentProvider) {
	aapmu.Lock()
	defer aapmu.Unlock()

	additionalAgentProviders = append(additionalAgentProviders, p)
	build.AgentProviders = append(build.AgentProviders, p.Version())
}

func (srv *Instance) setupAdditionalAgentProviders(ctx context.Context) error {
	aapmu.Lock()
	defer aapmu.Unlock()

	for _, provider := range additionalAgentProviders {
		provider.Initialize(srv.fw, srv.log)

		srv.log.Infof("Activating Agent Provider: %s", provider.Version())
		err := provider.RegisterAgents(ctx, srv.agents, srv.connector, srv.log)
		if err != nil {
			return err
		}
	}

	return nil
}
