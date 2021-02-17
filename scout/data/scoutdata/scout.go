package scoutdata

import (
	"context"
	"fmt"
	"time"

	"github.com/choria-io/go-choria/aagent/watchers/nagioswatcher"
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
	"github.com/choria-io/go-choria/providers/data"
	"github.com/choria-io/go-choria/providers/data/ddl"
	"github.com/choria-io/go-choria/providers/data/plugin"
	"github.com/choria-io/go-choria/server/agents"
)

type ScoutData struct{}

func ChoriaPlugin() *plugin.DataPlugin {
	return plugin.NewDataPlugin("scout", New)
}

func New(_ data.Framework) (data.Plugin, error) {
	return &ScoutData{}, nil
}

func (s *ScoutData) Run(_ context.Context, q data.Query, si agents.ServerInfoSource) (map[string]data.OutputItem, error) {
	query, ok := q.(string)
	if !ok {
		return nil, fmt.Errorf("could not parse query as a string")
	}

	machines, err := si.MachinesStatus()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve machine status: %s", err)
	}

	result := make(map[string]data.OutputItem)
	for _, m := range machines {
		if m.Scout && m.Name == query {
			result["name"] = m.Name
			result["version"] = m.Version
			result["state"] = m.State
			result["path"] = m.Path
			result["id"] = m.ID
			result["start_time"] = m.StartTimeUTC
			result["uptime"] = time.Now().Unix() - m.StartTimeUTC
			result["history"] = []string{}
			hist := []string{}
			n, ok := m.ScoutState.(*nagioswatcher.StateNotification)
			if ok {
				for _, h := range n.History {
					hist = append(hist, nagioswatcher.StateName(h.Status))
				}
			}
			result["history"] = hist

			break
		}
	}

	return result, nil
}

func (s *ScoutData) DLL() (*ddl.DDL, error) {
	sddl := &ddl.DDL{
		Metadata: ddl.Metadata{
			License:     "Apache-2.0",
			Author:      "R.I.Pienaar <rip@devco.net>",
			Timeout:     1,
			Name:        "scout",
			Version:     build.Version,
			URL:         "https://choria.io",
			Description: "Data about a specific Scout check",
			Provider:    "golang",
		},
		Query: &common.InputItem{
			Prompt:      "Check",
			Description: "The Scout check to retrieve data for",
			Type:        common.InputTypeString,
			Optional:    false,
			Validation:  "^[a-zA-Z][a-zA-Z0-9_-]+$",
			MaxLength:   50,
		},
		Output: map[string]*common.OutputItem{
			"name": {
				Description: "The name of the Scout check",
				DisplayAs:   "Name",
				Type:        common.OutputTypeString,
			},
			"version": {
				Description: "The version of the Scout check state machine",
				DisplayAs:   "Version",
				Type:        common.OutputTypeString,
			},
			"state": {
				Description: "The state the Scout check is in",
				DisplayAs:   "State",
				Type:        common.OutputTypeString,
			},
			"path": {
				Description: "The path on disk where the Scout check is stored",
				DisplayAs:   "Path",
				Type:        common.OutputTypeString,
			},
			"id": {
				Description: "The unique ID of the running state machine",
				DisplayAs:   "ID",
				Type:        common.OutputTypeString,
			},
			"start_time": {
				Description: "The time the check started in UTC",
				DisplayAs:   "Start Time",
				Type:        common.OutputTypeInteger,
			},
			"uptime": {
				Description: "The time the check has been running in seconds",
				DisplayAs:   "Uptime",
				Type:        common.OutputTypeInteger,
			},
			"history": {
				Description: "Recent past states of the check",
				DisplayAs:   "History",
				Type:        common.OutputTypeHash,
			},
		},
	}

	return sddl, nil
}
