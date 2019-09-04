package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
	Aggregation []ActionAggregateItem        `json:"aggregate"`
}

// ActionOutputItem describes an individual output item
type ActionOutputItem struct {
	Description string      `json:"description"`
	DisplayAs   string      `json:"display_as"`
	Default     interface{} `json:"default"`
}

// ActionAggregateItem describes a aggregate function to summarize data
type ActionAggregateItem struct {
	Function  string          `json:"function"`
	Arguments json.RawMessage `json:"args"`
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

// Find looks in the supplied libdirs for a DDL file for a specific agent
func Find(agent string, libdirs []string) (ddl *DDL, err error) {
	EachFile(libdirs, func(n string, f string) bool {
		if n == agent {
			ddl, err = New(f)
			return true
		}

		return false
	})

	if err != nil {
		return nil, fmt.Errorf("could not load agent %s: %s", agent, err)
	}

	if ddl == nil {
		return nil, fmt.Errorf("could not find DDL file for %s", agent)
	}

	return ddl, nil
}

// EachFile calls cb with a path to every found agent DDL, stops looking when br is true
func EachFile(libdirs []string, cb func(name string, path string) (br bool)) {
	for _, dir := range libdirs {
		agentsdir := filepath.Join(dir, "mcollective", "agent")

		filepath.Walk(agentsdir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			_, name := filepath.Split(path)
			extension := filepath.Ext(name)

			if extension != ".json" {
				return nil
			}

			cb(strings.TrimSuffix(name, extension), path)

			return nil
		})
	}
}

// ActionNames is a list of known actions defined by a DDL
func (d *DDL) ActionNames() []string {
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

// HaveAction determines if an action is known
func (d *DDL) HaveAction(action string) bool {
	_, err := d.ActionInterface(action)
	if err != nil {
		return false
	}

	return true
}

// Timeout is the timeout for this agent
func (d *DDL) Timeout() time.Duration {
	if d.Metadata.Timeout == 0 {
		return time.Duration(10 * time.Second)
	}

	return time.Duration(time.Second * time.Duration(d.Metadata.Timeout))
}
