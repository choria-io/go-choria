package inventory

import (
	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/protocol"
)

// DataSchema the schema of supported inventory files
const DataSchema = "https://choria.io/schemas/choria/discovery/v1/inventory_file.json"

// DataFile is a source for discovery information that describes a fleet
type DataFile struct {
	Schema string   `json:"$schema" yaml:"$schema"`
	Groups []*Group `json:"groups,omitempty" yaml:"groups,omitempty"`
	Nodes  []*Node  `json:"nodes" yaml:"nodes"`
}

type GroupFilter struct {
	Agents     []string `json:"agents" yaml:"agents"`
	Classes    []string `json:"classes" yaml:"classes"`
	Facts      []string `json:"facts" yaml:"facts"`
	Identities []string `json:"identities" yaml:"identities"`
	Compound   string   `json:"compound" yaml:"compound"`
}

func (f *GroupFilter) ToProtocolFilter() (*protocol.Filter, error) {
	filter := protocol.NewFilter()

	if f == nil {
		return filter, nil
	}

	for _, fact := range f.Facts {
		ff, err := facts.ParseFactFilterString(fact)
		if err != nil {
			return nil, err
		}

		err = filter.AddFactFilter(ff[0], ff[1], ff[2])
		if err != nil {
			return nil, err
		}
	}

	for _, agent := range f.Agents {
		filter.AddAgentFilter(agent)
	}

	for _, id := range f.Identities {
		filter.AddIdentityFilter(id)
	}

	for _, c := range f.Classes {
		filter.AddClassFilter(c)
	}

	if f.Compound != "" {
		err := filter.AddCompoundFilter(f.Compound)
		if err != nil {
			return nil, err
		}
	}

	return filter, nil
}

// Group is a view over the inventory expressed as a filter saved by name
type Group struct {
	Name   string       `json:"name" yaml:"name"`
	Filter *GroupFilter `json:"filter" yaml:"filter"`
}

// Agent describes an agent available on the node
type Agent struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
}

// Node describes a single node on the network
type Node struct {
	Name        string                 `json:"name" yaml:"name"`
	Collectives []string               `json:"collectives" yaml:"collectives"`
	Facts       map[string]interface{} `json:"facts" yaml:"facts"`
	Classes     []string               `json:"classes" yaml:"classes"`
	Agents      []*Agent               `json:"agents" yaml:"agents"`
}

// AgentNames returns the list of names for known agents
func (n *Node) AgentNames() []string {
	var names []string
	for _, a := range n.Agents {
		names = append(names, a.Name)
	}

	return names
}

// LookupGroup finds a group by name
func (d *DataFile) LookupGroup(name string) (*Group, bool) {
	for _, g := range d.Groups {
		if g.Name == name {
			return g, true
		}
	}

	return nil, false
}
