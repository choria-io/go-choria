package plugin

import (
	"fmt"
	"strings"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
)

func NewMachinePlugin(name string, machine interface{}) *MachinePlugin {
	return &MachinePlugin{name: name, machine: machine}
}

type MachinePlugin struct {
	name    string
	machine interface{}
}

func (p *MachinePlugin) Name() string {
	return p.name
}

func (p *MachinePlugin) Machine() interface{} {
	return p.machine
}

func (p *MachinePlugin) PluginInstance() interface{} {
	return p
}

func (p *MachinePlugin) PluginVersion() string {
	return build.Version
}

func (p *MachinePlugin) PluginName() string {
	return fmt.Sprintf("%s Autonomous Agent version %s", strings.Title(p.name), build.Version)
}

func (p *MachinePlugin) PluginType() inter.PluginType {
	return inter.MachinePlugin
}
