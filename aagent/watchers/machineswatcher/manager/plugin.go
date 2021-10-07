package machines_manager

import (
	_ "embed"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/plugin"
	"github.com/ghodss/yaml"
)

var (
	//go:embed machine.yaml
	mdat []byte
)

func ChoriaPlugin() *plugin.MachinePlugin {
	m := &machine.Machine{}
	err := yaml.Unmarshal(mdat, m)
	if err != nil {
		panic(err)
	}

	return plugin.NewMachinePlugin("machines_manager", m)
}
