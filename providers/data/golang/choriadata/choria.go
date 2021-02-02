package choriadata

import (
	"context"

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

	result := make(map[string]data.OutputItem)
	result["agents_count"] = len(si.KnownAgents())
	result["classes_count"] = len(si.Classes())
	result["machines_count"] = len(machines)
	result["config_file"] = si.ConfigFile()
	result["connected_broker"] = si.ConnectedServer()
	result["provisioning"] = si.Provisioning()
	result["uptime"] = si.UpTime()
	result["total_messages"] = stats.Total
	result["valid_messages"] = stats.Valid
	result["invalid_messages"] = stats.Invalid
	result["passed_messages"] = stats.Passed
	result["filtered_messages"] = stats.Filtered
	result["reply_messages"] = stats.Replies
	result["expired_messages"] = stats.TTLExpired

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
			"agents_count": {
				Description: "Number of active agents on the node",
				DisplayAs:   "Agents",
				Type:        "integer",
			},
			"classes_count": {
				Description: "Number of classes this node is tagged with",
				DisplayAs:   "Classes",
				Type:        "integer",
			},
			"machines_count": {
				Description: "The number of running Autonomous Agents",
				DisplayAs:   "Machines",
				Type:        "integer",
			},
			"config_file": {
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
			"total_messages": {
				Description: "The number of messages this server processed",
				DisplayAs:   "Total Messages",
				Type:        "integer",
			},
			"valid_messages": {
				Description: "The number of messages this server processed that passed validation",
				DisplayAs:   "Valid Messages",
				Type:        "integer",
			},
			"invalid_messages": {
				Description: "The number of messages this server processed that did not pass validation",
				DisplayAs:   "Invalid Messages",
				Type:        "integer",
			},
			"passed_messages": {
				Description: "The number of messages this server processed that matched filters",
				DisplayAs:   "Passed Messages",
				Type:        "integer",
			},
			"filtered_messages": {
				Description: "The number of messages this server processed that did not match filters",
				DisplayAs:   "Filtered Messages",
				Type:        "integer",
			},
			"reply_messages": {
				Description: "The number of reply messages this server sent",
				DisplayAs:   "Reply Messages",
				Type:        "integer",
			},
			"expired_messages": {
				Description: "The number of messages this server rejected due to TTL expiration",
				DisplayAs:   "Expired Messages",
				Type:        "integer",
			},
		},
	}

	return sddl, nil

}
