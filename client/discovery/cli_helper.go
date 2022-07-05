// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/filter"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/discovery/broadcast"
	"github.com/choria-io/go-choria/providers/discovery/external"
	"github.com/choria-io/go-choria/providers/discovery/flatfile"
	"github.com/choria-io/go-choria/providers/discovery/inventory"
	"github.com/choria-io/go-choria/providers/discovery/puppetdb"
	log "github.com/sirupsen/logrus"
)

type StandardOptions struct {
	Collective              string            `json:"collective"`
	FactFilter              []string          `json:"facts"`
	AgentFilter             []string          `json:"agents"`
	ClassFilter             []string          `json:"classes"`
	IdentityFilter          []string          `json:"identities"`
	CombinedFilter          []string          `json:"combined"`
	CompoundFilter          string            `json:"compound"`
	DiscoveryMethod         string            `json:"discovery_method"`
	DiscoveryTimeout        int               `json:"discovery_timeout"`
	DynamicDiscoveryTimeout bool              `json:"dynamic_discovery_timeout"`
	NodesFile               string            `json:"nodes_file"`
	DiscoveryOptions        map[string]string `json:"discovery_options"`

	unsetMethod bool
}

// NewStandardOptions creates a new CLI options helper
func NewStandardOptions() *StandardOptions {
	return &StandardOptions{
		FactFilter:       []string{},
		AgentFilter:      []string{},
		ClassFilter:      []string{},
		IdentityFilter:   []string{},
		CombinedFilter:   []string{},
		DiscoveryOptions: make(map[string]string),
	}
}

// Merge merges opt with the settings here, when a basic setting is
// set in opt it will overwrite the one here, when its a filter like
// a list or map it will extend ours with its values.
func (o *StandardOptions) Merge(opt *StandardOptions) {
	if opt.Collective != "" {
		o.Collective = opt.Collective
	}
	o.FactFilter = append(o.FactFilter, opt.FactFilter...)
	o.AgentFilter = append(o.AgentFilter, opt.AgentFilter...)
	o.ClassFilter = append(o.ClassFilter, opt.ClassFilter...)
	o.IdentityFilter = append(o.IdentityFilter, opt.IdentityFilter...)
	o.CombinedFilter = append(o.CombinedFilter, opt.CombinedFilter...)
	if opt.CompoundFilter != "" {
		o.CompoundFilter = opt.CompoundFilter
	}
	if opt.DiscoveryMethod != "" {
		o.DiscoveryMethod = opt.DiscoveryMethod
	}
	if opt.DiscoveryTimeout > 0 {
		o.DiscoveryTimeout = opt.DiscoveryTimeout
	}
	if opt.NodesFile != "" {
		o.NodesFile = opt.NodesFile
	}
	for k, v := range opt.DiscoveryOptions {
		o.DiscoveryOptions[k] = v
	}
}

// AddSelectionFlags adds the --dm and --discovery-timeout options
func (o *StandardOptions) AddSelectionFlags(app inter.FlagApp) {
	app.Flag("dm", "Sets a discovery method (mc, choria, file, external, inventory)").EnumVar(&o.DiscoveryMethod, "broadcast", "choria", "mc", "file", "flatfile", "external", "inventory")
	app.Flag("discovery-timeout", "Timeout for doing discovery").PlaceHolder("SECONDS").IntVar(&o.DiscoveryTimeout)
	app.Flag("discovery-window", "Enables a sliding window based dynamic discovery timeout (experimental)").UnNegatableBoolVar(&o.DynamicDiscoveryTimeout)
}

// AddFilterFlags adds the various flags like -W, -S, -T etc
func (o *StandardOptions) AddFilterFlags(app inter.FlagApp) {
	app.Flag("wf", "Match hosts with a certain fact").Short('F').StringsVar(&o.FactFilter)
	app.Flag("wc", "Match hosts with a certain configuration management class").Short('C').StringsVar(&o.ClassFilter)
	app.Flag("wa", "Match hosts with a certain Choria agent").Short('A').StringsVar(&o.AgentFilter)
	app.Flag("wi", "Match hosts with a certain Choria identity").Short('I').StringsVar(&o.IdentityFilter)
	app.Flag("with", "Combined classes and facts filter").Short('W').PlaceHolder("FILTER").StringsVar(&o.CombinedFilter)
	app.Flag("select", "Match hosts using a expr compound filter").Short('S').PlaceHolder("EXPR").StringVar(&o.CompoundFilter)
	app.Flag("target", "Target a specific sub collective").Short('T').StringVar(&o.Collective)
	app.Flag("do", "Options for the chosen discovery method").PlaceHolder("K=V").StringMapVar(&o.DiscoveryOptions)
}

// AddFlatFileFlags adds the flags to select nodes using --nodes in text, json and yaml formats
func (o *StandardOptions) AddFlatFileFlags(app inter.FlagApp) {
	app.Flag("nodes", "List of nodes to interact with in JSON, YAML or TEXT formats").ExistingFileVar(&o.NodesFile)
}

func (o *StandardOptions) Discover(ctx context.Context, fw inter.Framework, agent string, supportStdin bool, progress bool, logger *log.Entry) ([]string, time.Duration, error) {
	var (
		fformat    flatfile.SourceFormat
		sourceFile io.Reader
		nodes      []string
		to         = time.Second * time.Duration(o.DiscoveryTimeout)
	)

	filter, err := o.NewFilter(agent)
	if err != nil {
		return nil, 0, err
	}

	switch {
	case o.NodesFile != "":
		o.DiscoveryMethod = "flatfile"

		switch filepath.Ext(o.NodesFile) {
		case ".json":
			logger.Debugf("Using %q as JSON format file", o.NodesFile)
			fformat = flatfile.JSONFormat
		case ".yaml", ".yml":
			logger.Debugf("Using %q as YAML format file", o.NodesFile)
			fformat = flatfile.YAMLFormat
		default:
			logger.Debugf("Using %q as TEXT format file", o.NodesFile)
			fformat = flatfile.TextFormat
		}

		sourceFile, err = os.Open(o.NodesFile)
		if err != nil {
			return nil, 0, err
		}

	case len(filter.Compound) > 0 && o.DiscoveryMethod != "broadcast" && o.DiscoveryMethod != "inventory" && o.DiscoveryMethod != "mc":
		o.DiscoveryMethod = "broadcast"
		logger.Debugf("Forcing discovery mode to broadcast to support compound filters")

	case supportStdin && o.isPiped() && (o.DiscoveryMethod == "" || o.unsetMethod):
		stdin, err := ioutil.ReadAll(os.Stdin)
		stdin = bytes.TrimSpace(stdin)
		sourceFile = bytes.NewReader(stdin)

		if err != nil {
			logger.Debugf("Could not read STDIN to detect flatfile override")
			break
		}

		if len(stdin) == 0 {
			logger.Debugf("No data on STDIN found, not forcing flatfile override")
			break
		}

		if !(bytes.HasPrefix(stdin, []byte("{")) && bytes.HasSuffix(stdin, []byte("}"))) {
			logger.Debugf("Found non JSON data on STDIN, not forcing flatfile override")
			break
		}

		o.DiscoveryMethod = "flatfile"
		fformat = flatfile.ChoriaResponsesFormat
		logger.Debugf("Forcing discovery mode to flatfile with Choria responses on STDIN")
	}

	if o.DiscoveryMethod == "flatfile" && (fformat == 0 || sourceFile == nil) && len(o.DiscoveryOptions) == 0 {
		return nil, 0, fmt.Errorf("could not determine file to use as discovery source")
	}

	if progress {
		fmt.Printf("Discovering nodes using the %s method .... ", o.DiscoveryMethod)
	}

	start := time.Now()
	switch o.DiscoveryMethod {
	case "mc", "broadcast":
		opts := []broadcast.DiscoverOption{broadcast.Filter(filter), broadcast.Collective(o.Collective), broadcast.Timeout(to)}
		if o.DynamicDiscoveryTimeout {
			opts = append(opts, broadcast.SlidingWindow())
		}

		nodes, err = broadcast.New(fw).Discover(ctx, opts...)
	case "choria", "puppetdb":
		nodes, err = puppetdb.New(fw).Discover(ctx, puppetdb.Filter(filter), puppetdb.Collective(o.Collective), puppetdb.Timeout(to))
	case "external":
		nodes, err = external.New(fw).Discover(ctx, external.Filter(filter), external.Timeout(to), external.Collective(o.Collective), external.DiscoveryOptions(o.DiscoveryOptions))
	case "flatfile", "file":
		nodes, err = flatfile.New(fw).Discover(ctx, flatfile.Reader(sourceFile), flatfile.Format(fformat), flatfile.DiscoveryOptions(o.DiscoveryOptions))
	case "inventory":
		nodes, err = inventory.New(fw).Discover(ctx, inventory.Filter(filter), inventory.Collective(o.Collective), inventory.DiscoveryOptions(o.DiscoveryOptions))
	default:
		return nil, 0, fmt.Errorf("unsupported discovery method %q", o.DiscoveryMethod)
	}

	if progress {
		fmt.Printf("%d\n", len(nodes))
	}

	return nodes, time.Since(start), err
}

func (o *StandardOptions) isPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return (fi.Mode() & os.ModeCharDevice) == 0
}

// SetDefaultsFromChoria sets the defaults based on cfg
func (o *StandardOptions) SetDefaultsFromChoria(fw inter.Framework) {
	o.SetDefaultsFromConfig(fw.Configuration())
}

// SetDefaultsFromConfig sets the defaults based on cfg
func (o *StandardOptions) SetDefaultsFromConfig(cfg *config.Config) {
	if o.DiscoveryMethod == "" {
		o.DiscoveryMethod = cfg.DefaultDiscoveryMethod
		o.unsetMethod = true
	}

	if o.Collective == "" {
		o.Collective = cfg.MainCollective
	}

	if o.DiscoveryTimeout == 0 {
		o.DiscoveryTimeout = cfg.DiscoveryTimeout
	}

	if len(o.DiscoveryOptions) == 0 {
		for _, val := range cfg.DefaultDiscoveryOptions {
			parts := strings.Split(val, "=")
			if len(parts) == 2 {
				o.DiscoveryOptions[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
}

// NewFilter creates a new filter based on the options supplied, additionally agent will be added to the list
func (o *StandardOptions) NewFilter(agent string) (*protocol.Filter, error) {
	return filter.NewFilter(
		filter.FactFilter(o.FactFilter...),
		filter.AgentFilter(o.AgentFilter...),
		filter.ClassFilter(o.ClassFilter...),
		filter.IdentityFilter(o.IdentityFilter...),
		filter.CombinedFilter(o.CombinedFilter...),
		filter.CompoundFilter(o.CompoundFilter),
		filter.AgentFilter(agent),
	)
}
