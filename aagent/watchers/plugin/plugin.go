package plugin

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/plugin"
)

func NewWatcherPlugin(wtype string, version string, notification func() interface{}, new func(machine model.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (interface{}, error)) *WatcherPlugin {
	return &WatcherPlugin{
		Name: wtype,
		Creator: &creator{
			wtype:        wtype,
			version:      version,
			notification: notification,
			new:          new,
		},
	}
}

type creator struct {
	wtype        string
	version      string
	notification func() interface{}
	new          func(machine model.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (interface{}, error)
}

func (c *creator) Type() string {
	return c.wtype
}

func (c *creator) EventType() string {
	return fmt.Sprintf("io.choria.machine.watcher.%s.%s.state", c.wtype, c.version)
}

func (c *creator) UnmarshalNotification(n []byte) (interface{}, error) {
	state := c.notification()
	err := json.Unmarshal(n, state)

	return state, err
}

func (c *creator) New(machine model.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (interface{}, error) {
	return c.new(machine, name, states, failEvent, successEvent, interval, ai, properties)
}

type WatcherPlugin struct {
	Name    string
	Creator interface{}
}

// PluginInstance implements plugin.Pluggable
func (p *WatcherPlugin) PluginInstance() interface{} {
	return p.Creator
}

// PluginVersion implements plugin.Pluggable
func (p *WatcherPlugin) PluginVersion() string {
	return build.Version
}

// PluginName implements plugin.Pluggable
func (p *WatcherPlugin) PluginName() string {
	return fmt.Sprintf("%s Watcher version %s", strings.Title(p.Name), build.Version)
}

// PluginType implements plugin.Pluggable
func (p *WatcherPlugin) PluginType() plugin.Type {
	return plugin.MachineWatcherPlugin
}
