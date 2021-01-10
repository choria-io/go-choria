package cmd

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/client/discovery/broadcast"
	"github.com/choria-io/go-choria/client/discovery/puppetdb"
	"github.com/choria-io/go-choria/filter"
)

type discoverCommand struct {
	command

	jsonFormat       bool
	discoveryMethod  string
	discoveryTimeout int
	collective       string
	verbose          bool
	factF            []string
	agentsF          []string
	classF           []string
	identityF        []string
	combinedF        []string
	compoundF        string
}

func (d *discoverCommand) Setup() error {
	d.cmd = cli.app.Command("discover", "Discover nodes using the discovery system matching filter criteria").Alias("find")
	d.cmd.Flag("wf", "Match hosts with a certain fact").Short('F').StringsVar(&d.factF)
	d.cmd.Flag("wc", "Match hosts with a certain configuration management class").Short('C').StringsVar(&d.classF)
	d.cmd.Flag("wa", "Match hosts with a certain Choria agent").Short('A').StringsVar(&d.agentsF)
	d.cmd.Flag("wi", "Match hosts with a certain Choria identity").Short('I').StringsVar(&d.identityF)
	d.cmd.Flag("with", "Combined classes and facts filter").Short('W').PlaceHolder("FILTER").StringsVar(&d.combinedF)
	d.cmd.Flag("select", "Match hosts using a expr compound filter").Short('S').PlaceHolder("EXPR").StringVar(&d.compoundF)
	d.cmd.Flag("dm", "Sets a discovery method (broadcast, choria)").EnumVar(&d.discoveryMethod, "broadcast", "choria", "mc")
	d.cmd.Flag("target", "Target a specific sub collective").Short('T').StringVar(&d.collective)
	d.cmd.Flag("discovery-timeout", "Timeout for doing discovery").PlaceHolder("SECONDS").IntVar(&d.discoveryTimeout)
	d.cmd.Flag("verbose", "Log verbosely").Default("false").Short('v').BoolVar(&d.verbose)
	d.cmd.Flag("json", "Produce JSON output").Short('j').BoolVar(&d.jsonFormat)

	return nil
}

func (d *discoverCommand) Configure() error {
	return commonConfigure()
}

func (d *discoverCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if d.collective == "" {
		d.collective = cfg.MainCollective
	}

	if d.discoveryTimeout == 0 {
		d.discoveryTimeout = cfg.DiscoveryTimeout
	}

	if d.discoveryMethod == "" {
		d.discoveryMethod = cfg.DefaultDiscoveryMethod
	}

	filter, err := filter.NewFilter(
		filter.FactFilter(d.factF...),
		filter.AgentFilter(d.agentsF...),
		filter.ClassFilter(d.classF...),
		filter.IdentityFilter(d.identityF...),
		filter.CombinedFilter(d.combinedF...),
		filter.CompoundFilter(d.compoundF),
	)
	if err != nil {
		return err
	}

	if d.verbose && !d.jsonFormat {
		fmt.Printf("Discovering nodes using the %s method ....\n\n", d.discoveryMethod)
	}

	var nodes []string

	start := time.Now()
	switch d.discoveryMethod {
	case "mc", "broadcast":
		nodes, err = broadcast.New(c).Discover(ctx, broadcast.Filter(filter), broadcast.Collective(d.collective), broadcast.Timeout(time.Second*time.Duration(d.discoveryTimeout)))
	case "choria":
		nodes, err = puppetdb.New(c).Discover(ctx, puppetdb.Filter(filter), puppetdb.Collective(d.collective), puppetdb.Timeout(time.Second*time.Duration(d.discoveryTimeout)))
	}
	dt := time.Since(start)

	if d.jsonFormat {
		out, err := json.MarshalIndent(nodes, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	for _, n := range nodes {
		fmt.Println(n)
	}

	if d.verbose {
		fmt.Println()
		fmt.Printf("Discovered %d nodes using the %s method in %.02f seconds\n", len(nodes), d.discoveryMethod, dt.Seconds())
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &discoverCommand{})
}
