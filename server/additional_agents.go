package server

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

// AgentInitializer is a function signature for a function that can register an agent
type AgentInitializer func(context.Context, *agents.Manager, choria.InstanceConnector, *logrus.Entry) error

var additionalAgents []AgentInitializer
var aamu *sync.Mutex

func init() {
	additionalAgents = []AgentInitializer{}
	aamu = &sync.Mutex{}
}

// RegisterAdditionalAgent adds an agent to a running server
// this should be used for compile time injection of agents
// other than the ones that ship with choria
func RegisterAdditionalAgent(i AgentInitializer) {
	aamu.Lock()
	defer aamu.Unlock()

	additionalAgents = append(additionalAgents, i)
}

func (srv *Instance) setupAdditionalAgents(ctx context.Context) error {
	aamu.Lock()
	defer aamu.Unlock()

	for _, initializer := range additionalAgents {
		err := initializer(ctx, srv.agents, srv.connector, srv.log)
		if err != nil {
			return err
		}
	}

	return nil
}
