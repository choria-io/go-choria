package plugin

import (
	"fmt"

	"github.com/choria-io/go-choria/provtarget"
)

var provTargetResolverHost func(provtarget.TargetResolver) error

func init() {
	provTargetResolverHost = provtarget.RegisterTargetResolver
}

func registerProvisionTargetResolverPlugin(name string, plugin Pluggable) error {
	instance, ok := plugin.PluginInstance().(provtarget.TargetResolver)
	if !ok {
		return fmt.Errorf("plugin %s is not a valid ProvisionTargetResolver", plugin.PluginName())
	}

	provTargetResolverHost(instance)

	return nil
}
