package discovery

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/choria-io/go-choria/client/discovery/broadcast"
	"github.com/choria-io/go-choria/client/discovery/external"
	"github.com/choria-io/go-choria/client/discovery/flatfile"
	"github.com/choria-io/go-choria/client/discovery/puppetdb"
	"github.com/choria-io/go-choria/filter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/client"
)

type StandardOptions struct {
	Collective       string
	FactFilter       []string
	AgentFilter      []string
	ClassFilter      []string
	IdentityFilter   []string
	CombinedFilter   []string
	CompoundFilter   string
	DiscoveryMethod  string
	DiscoveryTimeout int
	NodesFile        string
}

// NewStandardOptions creates a new CLI options helper
func NewStandardOptions() *StandardOptions {
	return &StandardOptions{}
}

// AddSelectionFlags adds the --dm and --discovery-timeout options
func (o *StandardOptions) AddSelectionFlags(app *kingpin.CmdClause) {
	app.Flag("dm", "Sets a discovery method (mc, choria)").EnumVar(&o.DiscoveryMethod, "broadcast", "choria", "mc")
	app.Flag("discovery-timeout", "Timeout for doing discovery").PlaceHolder("SECONDS").IntVar(&o.DiscoveryTimeout)
}

// AddFilterFlags adds the various flags like -W, -S, -T etc
func (o *StandardOptions) AddFilterFlags(app *kingpin.CmdClause) {
	app.Flag("wf", "Match hosts with a certain fact").Short('F').StringsVar(&o.FactFilter)
	app.Flag("wc", "Match hosts with a certain configuration management class").Short('C').StringsVar(&o.ClassFilter)
	app.Flag("wa", "Match hosts with a certain Choria agent").Short('A').StringsVar(&o.AgentFilter)
	app.Flag("wi", "Match hosts with a certain Choria identity").Short('I').StringsVar(&o.IdentityFilter)
	app.Flag("with", "Combined classes and facts filter").Short('W').PlaceHolder("FILTER").StringsVar(&o.CombinedFilter)
	app.Flag("select", "Match hosts using a expr compound filter").Short('S').PlaceHolder("EXPR").StringVar(&o.CompoundFilter)
	app.Flag("target", "Target a specific sub collective").Short('T').StringVar(&o.Collective)
}

// AddFlatFileFlags adds the flags to select nodes using --nodes in text, json and yaml formats
func (o *StandardOptions) AddFlatFileFlags(app *kingpin.CmdClause) {
	app.Flag("nodes", "List of nodes to interact with in JSON, YAML or TEXT formats").ExistingFileVar(&o.NodesFile)
}

func (o *StandardOptions) Discover(ctx context.Context, fw client.ChoriaFramework, agent string, supportStdin bool, progress bool, logger *log.Entry) ([]string, time.Duration, error) {
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
	case len(filter.Compound) > 0:
		o.DiscoveryMethod = "broadcast"
		logger.Debugf("Forcing discovery mode to broadcast to support compound filters")

	case supportStdin && o.isPiped():
		o.DiscoveryMethod = "flatfile"
		fformat = flatfile.ChoriaResponses
		sourceFile = os.Stdin
		logger.Debugf("Forcing discovery mode to flatfile with Choria responses on STDIN")

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
	}

	if o.DiscoveryMethod == "flatfile" && (fformat == 0 || sourceFile == nil) {
		return nil, 0, fmt.Errorf("could not determine file to use as discovery source")
	}

	if progress {
		fmt.Printf("Discovering nodes using the %s method .... ", o.DiscoveryMethod)
	}

	start := time.Now()
	switch o.DiscoveryMethod {
	case "mc", "broadcast":
		nodes, err = broadcast.New(fw).Discover(ctx, broadcast.Filter(filter), broadcast.Collective(o.Collective), broadcast.Timeout(to))
	case "choria", "puppetdb":
		nodes, err = puppetdb.New(fw).Discover(ctx, puppetdb.Filter(filter), puppetdb.Collective(o.Collective), puppetdb.Timeout(to))
	case "external":
		nodes, err = external.New(fw).Discover(ctx, external.Filter(filter), external.Timeout(to), external.Collective(o.Collective))
	case "flatfile":
		nodes, err = flatfile.New(fw).Discover(ctx, flatfile.Reader(sourceFile), flatfile.Format(fformat))
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

// SetDefaults sets default values for options, should be called before doing any discovery after flags are parsed
func (o *StandardOptions) SetDefaults(collective string, dm string, dt int) {
	if o.DiscoveryMethod == "" {
		o.DiscoveryMethod = dm
	}

	if o.Collective == "" {
		o.Collective = collective
	}

	if o.DiscoveryTimeout == 0 {
		o.DiscoveryTimeout = dt
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
