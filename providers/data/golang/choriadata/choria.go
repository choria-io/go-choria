package choriadata

import (
	"context"
	"os"
	"runtime"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
	"github.com/choria-io/go-choria/providers/data"
	"github.com/choria-io/go-choria/providers/data/ddl"
	"github.com/choria-io/go-choria/providers/data/plugin"
	"github.com/choria-io/go-choria/server/agents"
)

type ChoriaData struct{}

func ChoriaPlugin() *plugin.DataPlugin {
	return plugin.NewDataPlugin("choria", New)
}

func New(_ data.Framework) (data.Plugin, error) {
	return &ChoriaData{}, nil
}

func (s *ChoriaData) Run(_ context.Context, q data.Query, si agents.ServerInfoSource) (map[string]data.OutputItem, error) {
	machines, _ := si.MachinesStatus()
	stats := si.Stats()
	classes := si.Classes()
	agents := si.KnownAgents()
	machineNames := []string{}

	for _, m := range machines {
		machineNames = append(machineNames, m.Name)
	}

	result := map[string]data.OutputItem{
		"agents":           agents,
		"agents_count":     len(agents),
		"built":            si.BuildInfo().BuildDate(),
		"classes":          classes,
		"classes_count":    len(classes),
		"commit":           si.BuildInfo().SHA(),
		"configfile":       si.ConfigFile(),
		"connected_broker": si.ConnectedServer(),
		"cpus":             runtime.NumCPU(),
		"filtered":         stats.Filtered,
		"go_version":       runtime.Version(),
		"goroutines":       runtime.NumGoroutine(),
		"license":          si.BuildInfo().License(),
		"machines_count":   len(machines),
		"machines":         machineNames,
		"passed":           stats.Passed,
		"pid":              os.Getpid(),
		"provisioning":     si.Provisioning(),
		"replies":          stats.Replies,
		"total":            stats.Total,
		"ttlexpired":       stats.TTLExpired,
		"unvalidated":      stats.Invalid,
		"uptime":           si.UpTime(),
		"validated":        stats.Valid,
		"version":          si.BuildInfo().Version(),
	}

	return result, nil
}

func (s *ChoriaData) DLL() (*ddl.DDL, error) {
	sddl := &ddl.DDL{
		Metadata: ddl.Metadata{
			License:     "Apache-2.0",
			Author:      "R.I.Pienaar <rip@devco.net>",
			Timeout:     1,
			Name:        "choria",
			Version:     build.Version,
			URL:         "https://choria.io",
			Description: "Data about a the running Choria instance",
			Provider:    "golang",
		},
		Output: map[string]*common.OutputItem{
			"agents": {
				Description: "Known agents hosted by this server",
				DisplayAs:   "Agents",
				Type:        common.OutputTypeArray,
			},
			"pid": {
				Description: "The process ID of the running process",
				DisplayAs:   "PID",
				Type:        common.OutputTypeInteger,
			},
			"agents_count": {
				Description: "Number of active agents on the node",
				DisplayAs:   "Agents",
				Type:        common.OutputTypeInteger,
			},
			"classes": {
				Description: "List of classes this machine is tagged with",
				DisplayAs:   "Class Names",
				Type:        common.OutputTypeArray,
			},
			"classes_count": {
				Description: "Number of classes this node is tagged with",
				DisplayAs:   "Classes",
				Type:        common.OutputTypeInteger,
			},
			"machines": {
				Description: "The names of running machines",
				DisplayAs:   "Machines Names",
				Type:        common.OutputTypeArray,
			},
			"machines_count": {
				Description: "The number of running Autonomous Agents",
				DisplayAs:   "Machines",
				Type:        common.OutputTypeInteger,
			},
			"configfile": {
				Description: "The path to the running configuration",
				DisplayAs:   "Configuration File",
				Type:        common.OutputTypeString,
			},
			"connected_broker": {
				Description: "The Choria Broker this server is connected to",
				DisplayAs:   "Connected Broker",
				Type:        common.OutputTypeString,
			},
			"provisioning": {
				Description: "If the node is currently in Provisioning mode",
				DisplayAs:   "Provisioning",
				Type:        common.OutputTypeBoolean,
			},
			"uptime": {
				Description: "The time, in seconds, that the server has been up",
				DisplayAs:   "Uptime",
				Type:        common.OutputTypeInteger,
			},
			"total": {
				Description: "The number of messages this server processed",
				DisplayAs:   "Total Messages",
				Type:        common.OutputTypeInteger,
			},
			"validated": {
				Description: "The number of messages this server processed that passed validation",
				DisplayAs:   "Valid Messages",
				Type:        common.OutputTypeInteger,
			},
			"unvalidated": {
				Description: "The number of messages this server processed that did not pass validation",
				DisplayAs:   "Invalid Messages",
				Type:        common.OutputTypeInteger,
			},
			"passed": {
				Description: "The number of messages this server processed that matched filters",
				DisplayAs:   "Passed Messages",
				Type:        common.OutputTypeInteger,
			},
			"filtered": {
				Description: "The number of messages this server processed that did not match filters",
				DisplayAs:   "Filtered Messages",
				Type:        common.OutputTypeInteger,
			},
			"replies": {
				Description: "The number of reply messages this server sent",
				DisplayAs:   "Reply Messages",
				Type:        common.OutputTypeInteger,
			},
			"ttlexpired": {
				Description: "The number of messages this server rejected due to TTL expiration",
				DisplayAs:   "Expired Messages",
				Type:        common.OutputTypeInteger,
			},
			"version": {
				Description: "The running version of the server",
				DisplayAs:   "Version",
				Type:        common.OutputTypeString,
			},
			"go_version": {
				Description: "Version of Go used to build the server",
				DisplayAs:   "Golang",
				Type:        common.OutputTypeString,
			},
			"goroutines": {
				Description: "The number of active Go Routines in the server process",
				DisplayAs:   "Go Routines",
				Type:        common.OutputTypeInteger,
			},
			"cpus": {
				Description: "The number of logical CPUs available to the Go runtime",
				DisplayAs:   "CPUs",
				Type:        common.OutputTypeInteger,
			},
			"license": {
				Description: "The license this binary is released under",
				DisplayAs:   "License",
				Type:        common.OutputTypeString,
			},
			"built": {
				Description: "The time when the build was performed",
				DisplayAs:   "Built",
				Type:        common.OutputTypeString,
			},
			"commit": {
				Description: "The source commit used to build this instance",
				DisplayAs:   "Commit",
				Type:        common.OutputTypeString,
			},
		},
	}

	return sddl, nil

}
