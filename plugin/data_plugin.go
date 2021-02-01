package plugin

import (
	"github.com/choria-io/go-choria/providers/data"
)

func registerDataPlugin(_ string, plugin Pluggable) error {
	instance := plugin.PluginInstance().(*data.Creator)
	// if !ok {
	// 	return fmt.Errorf("%s is not a valid data plugin", plugin.PluginName())
	// }

	return data.RegisterPlugin(plugin.PluginName(), instance)

}
