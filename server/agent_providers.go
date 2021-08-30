package server

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"
)

// AgentProvider is capable of adding agents into a running instance
type AgentProvider interface {
	Initialize(cfg *config.Config, log *logrus.Entry)
	RegisterAgents(ctx context.Context, mgr AgentManager, connector inter.AgentConnector, log *logrus.Entry) error
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
	util.BuildInfo().RegisterAgentProvider(p.Version())
}

func (srv *Instance) setupAdditionalAgentProviders(ctx context.Context) error {
	aapmu.Lock()
	providers := make([]AgentProvider, len(additionalAgentProviders))
	copy(providers, additionalAgentProviders)
	aapmu.Unlock()

	for _, provider := range providers {
		provider.Initialize(srv.fw.Configuration(), srv.log)

		srv.log.Infof("Activating Agent Provider: %s", provider.Version())
		err := provider.RegisterAgents(ctx, srv.agents, srv.connector, srv.log)
		if err != nil {
			return err
		}
	}

	return nil
}
