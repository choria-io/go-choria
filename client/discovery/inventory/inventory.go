package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
)

// DataSchema the schema of supported inventory files
const DataSchema = "https://choria.io/schemas/choria/discovery/v1/inventory_file.json"

type Inventory struct {
	fw  ChoriaFramework
	log *logrus.Entry
}

// DataFile is a source for discovery information that describes a fleet
type DataFile struct {
	Schema string   `json:"$schema" yaml:"$schema"`
	Groups []*Group `json:"groups,omitempty" yaml:"groups,omitempty"`
	Nodes  []*Node  `json:"nodes" yaml:"nodes"`
}

type GroupFilter struct {
	Facts      []string `json:"facts" yaml:"facts"`
	Agents     []string `json:"agents" yaml:"agents"`
	Identities []string `json:"identities" yaml:"identities"`
	Classes    []string `json:"classes" yaml:"classes"`
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

type ChoriaFramework interface {
	Logger(string) *logrus.Entry
	Configuration() *config.Config
}

// New creates a new puppetdb discovery client
func New(fw ChoriaFramework) *Inventory {
	b := &Inventory{
		fw:  fw,
		log: fw.Logger("inventory_discovery"),
	}

	return b
}

// Discover performs a broadcast discovery using the supplied filter
func (i *Inventory) Discover(_ context.Context, opts ...DiscoverOption) (n []string, err error) {
	dopts := &dOpts{
		collective: i.fw.Configuration().MainCollective,
		source:     i.fw.Configuration().Choria.InventoryDiscoverySource,
		filter:     protocol.NewFilter(),
		do:         make(map[string]string),
	}

	for _, opt := range opts {
		opt(dopts)
	}

	file, ok := dopts.do["file"]
	if ok {
		dopts.source = file
	}

	if dopts.source == "" {
		return nil, fmt.Errorf("no discovery source file specified")
	}

	if !util.FileExist(dopts.source) {
		return nil, fmt.Errorf("discovery source %q does not exist", dopts.source)
	}

	if len(dopts.filter.CompoundFilters()) > 0 {
		return nil, fmt.Errorf("compound filters are not supported")
	}

	return i.discover(dopts)
}

func (i *Inventory) discover(dopts *dOpts) ([]string, error) {
	data, err := i.readInventory(dopts.source)
	if err != nil {
		return nil, err
	}

	grouped, err := i.isValidGroupLookup(dopts.filter)
	if err != nil {
		return nil, err
	}

	if grouped {
		matched := []string{}
		for _, id := range dopts.filter.IdentityFilters() {
			id := strings.TrimPrefix(id, "group:")
			grp, ok := data.LookupGroup(id)
			if ok {
				gf, err := grp.Filter.ToProtocolFilter()
				if err != nil {
					return nil, err
				}
				selected, err := i.selectMatchingNodes(data, "", gf)
				if err != nil {
					return nil, err
				}

				matched = append(matched, selected...)
			} else {
				return nil, fmt.Errorf("unknown group '%s'", id)
			}
		}

		return util.UniqueStrings(matched, true), nil
	}

	return i.selectMatchingNodes(data, dopts.collective, dopts.filter)
}

func (i *Inventory) isValidGroupLookup(f *protocol.Filter) (grouped bool, err error) {
	grp := 0
	node := 0

	idf := len(f.IdentityFilters())
	if idf > 0 {
		for _, f := range f.IdentityFilters() {
			if strings.HasPrefix(f, "group:") {
				grp++
			} else {
				node++
			}
		}
	}

	if grp == 0 {
		return false, nil
	}

	if node != 0 {
		return true, fmt.Errorf("group matches cannot be combined with other filters")
	}

	if len(f.FactFilters()) > 0 || len(f.ClassFilters()) > 0 || len(f.AgentFilters()) > 0 || len(f.CompoundFilters()) > 0 {
		return true, fmt.Errorf("group matches cannot be combined with other filters")
	}

	return true, nil
}

func (i *Inventory) selectMatchingNodes(d *DataFile, collective string, f *protocol.Filter) ([]string, error) {
	var matched []string

	for _, node := range d.Nodes {
		if collective != "" && !util.StringInList(node.Collectives, collective) {
			continue
		}

		if f.Empty() {
			matched = append(matched, node.Name)
			continue
		}

		passed := 0

		if len(f.IdentityFilters()) > 0 {
			if f.MatchIdentity(node.Name) {
				passed++
			} else {
				continue
			}
		}

		if len(f.AgentFilters()) > 0 {
			if f.MatchAgents(node.AgentNames()) {
				passed++
			} else {
				continue
			}
		}

		if len(f.ClassFilters()) > 0 {
			if f.MatchClasses(node.Classes, i.log) {
				passed++
			} else {
				continue
			}
		}

		if len(f.FactFilters()) > 0 {
			fj, err := json.Marshal(node.Facts)
			if err != nil {
				return nil, fmt.Errorf("invalid facts: %s", err)
			}
			if f.MatchFacts(fj, i.log) {
				passed++
			} else {
				continue
			}
		}

		if passed > 0 {
			matched = append(matched, node.Name)
		}
	}

	return matched, nil
}

func (i *Inventory) readInventory(path string) (*DataFile, error) {
	ext := filepath.Ext(path)
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data := &DataFile{}

	if ext == ".yaml" || ext == ".yml" {
		f, err = yaml.YAMLToJSON(f)
		if err != nil {
			return nil, err
		}
	}

	err = json.Unmarshal(f, data)
	if err != nil {
		return nil, err
	}

	if data.Schema != DataSchema {
		return nil, fmt.Errorf("invalid schema %q expected %q", data.Schema, DataSchema)
	}

	return data, nil
}
