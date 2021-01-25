package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
)

type Inventory struct {
	fw  ChoriaFramework
	log *logrus.Entry
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

	if len(f.FactFilters()) > 0 || len(f.ClassFilters()) > 0 || (len(f.AgentFilters()) > 0 && !reflect.DeepEqual(f.AgentFilters(), []string{"rpcutil"})) || len(f.CompoundFilters()) > 0 {
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

		agents := node.AgentNames()
		if len(f.AgentFilters()) > 0 {
			if f.MatchAgents(agents) {
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

		fj, err := json.Marshal(node.Facts)
		if err != nil {
			return nil, fmt.Errorf("invalid facts: %s", err)
		}

		if len(f.FactFilters()) > 0 {
			if f.MatchFacts(fj, i.log) {
				passed++
			} else {
				continue
			}
		}

		if len(f.CompoundFilters()) > 0 {
			if f.MatchCompound(fj, node.Classes, agents, i.log) {
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

	err = ValidateInventory(f)
	if err != nil {
		return nil, err
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
