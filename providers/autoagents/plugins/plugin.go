// Copyright (c) 2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"fmt"
	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/plugin"
	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
)

func init() {
	bi := &build.Info{}
	bi.RegisterMachine(fmt.Sprintf("Autonomous Agent Plugins Manager version %s", build.Version))
}

func ChoriaPlugin(cfg *config.Config) (*plugin.MachinePlugin, error) {
	if !cfg.Choria.AutonomousAgentsDownload {
		return nil, fmt.Errorf("autonomous agent plugin management is not enabled")
	}

	bucket := cfg.Choria.AutonomousAgentsBucket
	key := cfg.Choria.AutonomousAgentsKey
	purgeUnknown := cfg.Choria.AutonomousAgentsPurge
	bucketInterval := cfg.Choria.AutonomousAgentsBucketPollInterval
	checkInterval := cfg.Choria.AutonomousAgentCheckInterval
	pluginsDirectory := cfg.Choria.MachineSourceDir
	publicKey := cfg.Choria.AutonomousAgentPublicKey

	m := machine.Machine{
		MachineName:    "plugins_manager",
		InitialState:   "INITIAL",
		MachineVersion: build.Version,
		Transitions: []*machine.Transition{
			{
				Name:        "enter_maintenance",
				From:        []string{"MANAGE"},
				Destination: "MAINTENANCE",
				Description: "Stops actively managing plugins",
			},
			{
				Name:        "resume",
				From:        []string{"MAINTENANCE"},
				Destination: "MANAGE",
				Description: "Resume normal operations after being in maintenance mode",
			},
			{
				Name:        "manage_plugins",
				From:        []string{"INITIAL"},
				Destination: "MANAGE",
				Description: "Actively manage plugins",
			},
		},
		WatcherDefs: []*watchers.WatcherDef{
			{
				Name:              "initial_specification",
				Type:              "kv",
				Interval:          bucketInterval,
				StateMatch:        []string{"INITIAL"},
				SuccessTransition: "manage_plugins",
				Properties: map[string]any{
					"bucket":            bucket,
					"key":               key,
					"mode":              "poll",
					"bucket_prefix":     false,
					"on_successful_get": true,
				},
			},
			{
				Name:       "specification",
				Type:       "kv",
				Interval:   bucketInterval,
				StateMatch: []string{"MANAGE"},
				Properties: map[string]any{
					"bucket":        bucket,
					"key":           key,
					"mode":          "poll",
					"bucket_prefix": false,
				},
			},
			{
				Name:       "manage_machines",
				StateMatch: []string{"MANAGE"},
				Type:       "plugins",
				Interval:   checkInterval,
				Properties: map[string]any{
					"data_item":               key,
					"purge_unknown":           purgeUnknown,
					"manager_machine_prefix":  "mm",
					"plugin_manage_interval":  checkInterval,
					"machine_manage_interval": checkInterval,
					"public_key":              publicKey,
					"plugins_directory":       pluginsDirectory,
				},
			},
		},
	}

	return plugin.NewMachinePlugin(m.MachineName, &m), nil
}
