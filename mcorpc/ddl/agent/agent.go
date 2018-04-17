package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/choria-io/go-choria/server/agents"
)

// DDL represents the schema of a mcorpc agent and is at a basic level
// compatible with the mcollective agent ddl format
type DDL struct {
	Schema   string           `json:"$schema"`
	Metadata *agents.Metadata `json:"metadata"`
	Actions  []*Action        `json:"actions"`
}

// Action describes an individual action in an agent
type Action struct {
	Name        string                       `json:"action"`
	Input       json.RawMessage              `json:"input"`
	Output      map[string]*ActionOutputItem `json:"output"`
	Display     string                       `json:"display"`
	Description string                       `json:"description"`
}

// ActionOutputItem describes an individual output item
type ActionOutputItem struct {
	Description string      `json:"description"`
	DisplayAs   string      `json:"display_as"`
	Default     interface{} `json:"default"`
}

// New creates a new DDL from a JSON file
func New(file string) (*DDL, error) {
	ddl := &DDL{}

	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not load DDL data: %s", err)
	}

	err = json.Unmarshal(dat, ddl)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON data in %s: %s", file, err)
	}

	return ddl, nil
}

// ActionList is a list of known actions defined by a DDL
func (d *DDL) ActionList() []string {
	actions := []string{}

	if d.Actions != nil {
		for _, act := range d.Actions {
			actions = append(actions, act.Name)
		}
	}

	return actions
}

// ActionInterface looks up an Action by name
func (d *DDL) ActionInterface(action string) (*Action, error) {
	for _, act := range d.Actions {
		if act.Name == action {
			return act, nil
		}
	}

	return nil, fmt.Errorf("could not found an action called %s#%s", d.Metadata.Name, action)
}

// Timeout is the timeout for this agent
func (d *DDL) Timeout() time.Duration {
	if d.Metadata.Timeout == 0 {
		return time.Duration(10 * time.Second)
	}

	return time.Duration(time.Second * time.Duration(d.Metadata.Timeout))
}
