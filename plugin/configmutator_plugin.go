package plugin

import (
	"fmt"

	"github.com/choria-io/go-config"
)

func registerConfigMutator(name string, plugin Pluggable) error {
	mutator, ok := plugin.PluginInstance().(config.Mutator)
	if !ok {
		return fmt.Errorf("%s is not a valid configuration mutator plugin", plugin.PluginName())
	}

	config.RegisterMutator(plugin.PluginName(), mutator)

	return nil
}
