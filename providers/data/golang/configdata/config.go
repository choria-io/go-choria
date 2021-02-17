package configdata

import (
	"context"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/confkey"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
	"github.com/choria-io/go-choria/providers/data"
	"github.com/choria-io/go-choria/providers/data/ddl"
	"github.com/choria-io/go-choria/providers/data/plugin"
	"github.com/choria-io/go-choria/server/agents"
)

type ConfigData struct {
	cfg *config.Config
}

func ChoriaPlugin() *plugin.DataPlugin {
	return plugin.NewDataPlugin("config_item", New)
}

func New(fw data.Framework) (data.Plugin, error) {
	return &ConfigData{cfg: fw.Configuration()}, nil
}

func (s *ConfigData) Run(_ context.Context, q data.Query, si agents.ServerInfoSource) (map[string]data.OutputItem, error) {
	item := q.(string)
	val, ok := confkey.InterfaceWithKey(s.cfg, item)
	if !ok {
		val, ok = confkey.InterfaceWithKey(s.cfg.Choria, item)
		if !ok {
			return map[string]data.OutputItem{
				"present": false,
				"value":   nil,
			}, nil
		}
	}

	return map[string]data.OutputItem{
		"present": true,
		"value":   val,
	}, nil
}

func (s *ConfigData) DLL() (*ddl.DDL, error) {
	sddl := &ddl.DDL{
		Metadata: ddl.Metadata{
			License:     "Apache-2.0",
			Author:      "R.I.Pienaar <rip@devco.net>",
			Timeout:     1,
			Name:        "config_item",
			Version:     build.Version,
			URL:         "https://choria.io",
			Description: "Runtime value of a configuration items",
			Provider:    "golang",
		},
		Query: &common.InputItem{
			Prompt:      "Configuration Key",
			Description: "A key as found in the configuration file",
			Type:        common.InputTypeString,
			Default:     "",
			Optional:    false,
			Validation:  ".+",
			MaxLength:   256,
		},
		Output: map[string]*common.OutputItem{
			"present": {
				Description: "Indicates if the configuration item is known",
				DisplayAs:   "Present",
				Type:        common.OutputTypeBoolean,
			},
			"value": {
				Description: "The current value of the configuration item, in it's native data type",
				DisplayAs:   "Value",
				Type:        common.OutputTypeAny,
			},
		},
	}

	return sddl, nil
}
