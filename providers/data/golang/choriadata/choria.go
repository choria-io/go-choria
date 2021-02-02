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
				Type:        "array",
			},
			"pid": {
				Description: "The process ID of the running process",
				DisplayAs:   "PID",
				Type:        "integer",
			},
			"agents_count": {
				Description: "Number of active agents on the node",
				DisplayAs:   "Agents",
				Type:        "integer",
			},
			"classes": {
				Description: "List of classes this machine is tagged with",
				DisplayAs:   "Class Names",
				Type:        "array",
			},
			"classes_count": {
				Description: "Number of classes this node is tagged with",
				DisplayAs:   "Classes",
				Type:        "integer",
			},
			"machines": {
				Description: "The names of running machines",
				DisplayAs:   "Machines Names",
				Type:        "array",
			},
			"machines_count": {
				Description: "The number of running Autonomous Agents",
				DisplayAs:   "Machines",
				Type:        "integer",
			},
			"configfile": {
				Description: "The path to the running configuration",
				DisplayAs:   "Configuration File",
				Type:        "string",
			},
			"connected_broker": {
				Description: "The Choria Broker this server is connected to",
				DisplayAs:   "Connected Broker",
				Type:        "string",
			},
			"provisioning": {
				Description: "If the node is currently in Provisioning mode",
				DisplayAs:   "Provisioning",
				Type:        "bool",
			},
			"uptime": {
				Description: "The time, in seconds, that the server has been up",
				DisplayAs:   "Uptime",
				Type:        "integer",
			},
			"total": {
				Description: "The number of messages this server processed",
				DisplayAs:   "Total Messages",
				Type:        "integer",
			},
			"validated": {
				Description: "The number of messages this server processed that passed validation",
				DisplayAs:   "Valid Messages",
				Type:        "integer",
			},
			"unvalidated": {
				Description: "The number of messages this server processed that did not pass validation",
				DisplayAs:   "Invalid Messages",
				Type:        "integer",
			},
			"passed": {
				Description: "The number of messages this server processed that matched filters",
				DisplayAs:   "Passed Messages",
				Type:        "integer",
			},
			"filtered": {
				Description: "The number of messages this server processed that did not match filters",
				DisplayAs:   "Filtered Messages",
				Type:        "integer",
			},
			"replies": {
				Description: "The number of reply messages this server sent",
				DisplayAs:   "Reply Messages",
				Type:        "integer",
			},
			"ttlexpired": {
				Description: "The number of messages this server rejected due to TTL expiration",
				DisplayAs:   "Expired Messages",
				Type:        "integer",
			},
			"version": {
				Description: "The running version of the server",
				DisplayAs:   "Version",
				Type:        "string",
			},
			"go_version": {
				Description: "Version of Go used to build the server",
				DisplayAs:   "Golang",
				Type:        "string",
			},
			"goroutines": {
				Description: "The number of active Go Routines in the server process",
				DisplayAs:   "Go Routines",
				Type:        "integer",
			},
			"cpus": {
				Description: "The number of logical CPUs available to the Go runtime",
				DisplayAs:   "CPUs",
				Type:        "integer",
			},
			"license": {
				Description: "The license this binary is relased under",
				DisplayAs:   "License",
				Type:        "string",
			},
			"built": {
				Description: "The time when the build was performed",
				DisplayAs:   "Built",
				Type:        "string",
			},
			"commit": {
				Description: "The source commit used to build this instance",
				DisplayAs:   "Commit",
				Type:        "string",
			},
		},
	}

	return sddl, nil

}
