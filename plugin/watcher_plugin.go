package plugin

import (
	"fmt"

	"github.com/choria-io/go-choria/aagent/watchers"
)

func registerWatcherPlugin(_ string, plugin Pluggable) error {
	instance, ok := plugin.PluginInstance().(watchers.WatcherConstructor)
	if !ok {
		return fmt.Errorf("%s is not a valid watcher plugin", plugin.PluginName())
	}

	return watchers.RegisterWatcherPlugin(plugin.PluginName(), instance)
}
