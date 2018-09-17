package plugin

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

var agentHost func(server.AgentInitializer)

func init() {
	agentHost = server.RegisterAdditionalAgent
}

func registerAgentPlugin(name string, plugin Pluggable) error {
	instance, ok := plugin.PluginInstance().(func(server.AgentManager) (agents.Agent, error))
	if !ok {
		return fmt.Errorf("%s is not a valid agent plugin", plugin.PluginName())
	}

	initializer := func(ctx context.Context, mgr *agents.Manager, connector choria.InstanceConnector, log *logrus.Entry) error {
		log.Infof("Registering additional agent %s version %s", name, plugin.PluginVersion())

		a, err := instance(mgr)
		if err != nil {
			return fmt.Errorf("could not create %s agent: %s", name, err)
		}

		mgr.RegisterAgent(ctx, name, a, connector)

		return nil
	}

	agentHost(initializer)

	return nil
}
