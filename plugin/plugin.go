package plugin

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/fs"
	"github.com/ghodss/yaml"
)

// List is a list of plugins to load
type List struct {
	Plugins []*Plugin
}

// Plugin is an individual plugin
type Plugin struct {
	Name string
	Repo string
}

// Pluggable is a Choria Plugin
type Pluggable interface {
	// PluginInstance is any structure that implements the plugin, should be right type for the kind of plugin
	PluginInstance() interface{}

	// PluginName is a human friendly name for the plugin
	PluginName() string

	// PluginType is the type of the plugin, to match inter.PluginType
	PluginType() inter.PluginType

	// PluginVersion is the version of the plugin
	PluginVersion() string
}

// Register registers a type of plugin into the choria server
func Register(name string, plugin Pluggable) error {
	var err error

	switch inter.PluginType(plugin.PluginType()) {
	case inter.AgentProviderPlugin:
		err = registerAgentProviderPlugin(name, plugin)

	case inter.AgentPlugin:
		err = registerAgentPlugin(name, plugin)

	case inter.ProvisionTargetResolverPlugin:
		err = registerProvisionTargetResolverPlugin(name, plugin)

	case inter.ConfigMutatorPlugin:
		err = registerConfigMutator(name, plugin)

	case inter.MachineWatcherPlugin:
		err = registerWatcherPlugin(name, plugin)

	case inter.DataPlugin:
		err = registerDataPlugin(name, plugin)

	default:
		err = fmt.Errorf("unknown plugin type %d from %s", plugin.PluginType(), name)
	}

	return err
}

// Load loads a plugin list from file
func Load(file string) (*List, error) {
	rawdat := make(map[string]string)
	input, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(input, &rawdat)
	if err != nil {
		return nil, fmt.Errorf("could not parse yaml: %s", err)
	}

	list := &List{Plugins: []*Plugin{}}
	for k, v := range rawdat {
		list.Plugins = append(list.Plugins, &Plugin{Name: k, Repo: v})
	}

	sort.Slice(list.Plugins, func(i, j int) bool {
		return list.Plugins[i].Name < list.Plugins[j].Name
	})

	return list, err
}

// Now is the current time
func (p *Plugin) Now() string {
	return time.Now().String()
}

// Loader is the loader go code
func (p *Plugin) Loader() (string, error) {
	out, err := fs.ExecuteTemplate("plugin/plugin.templ", p, nil)
	if err != nil {
		return "", err
	}

	return string(out), err
}
