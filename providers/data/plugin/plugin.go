package plugin

import (
	"fmt"
	"strings"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/data"
)

type DataPlugin struct {
	Name    string
	Creator *data.Creator
}

func NewDataPlugin(name string, creator func(fw data.Framework) (data.Plugin, error)) *DataPlugin {
	return &DataPlugin{Name: name, Creator: &data.Creator{Name: name, F: creator}}
}

// PluginInstance implements plugin.Pluggable
func (p *DataPlugin) PluginInstance() interface{} {
	return p.Creator
}

// PluginVersion implements plugin.Pluggable
func (p *DataPlugin) PluginVersion() string {
	return build.Version
}

// PluginName implements plugin.Pluggable
func (p *DataPlugin) PluginName() string {
	return fmt.Sprintf("%s Data version %s", strings.Title(p.Name), build.Version)
}

// PluginType implements plugin.Pluggable
func (p *DataPlugin) PluginType() inter.PluginType {
	return inter.DataPlugin
}
