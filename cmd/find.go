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

type findCommand struct {
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
}

func (f *findCommand) Setup() error {
	f.cmd = cli.app.Command("find", "Find hosts using the discovery system matching filter criteria")
	f.cmd.Flag("wf", "Match hosts with a certain fact").Short('F').StringsVar(&f.factF)
	f.cmd.Flag("wc", "Match hosts with a certain configuration management class").Short('C').StringsVar(&f.classF)
	f.cmd.Flag("wa", "Match hosts with a certain Choria agent").Short('A').StringsVar(&f.agentsF)
	f.cmd.Flag("wi", "Match hosts with a certain Choria identity").Short('I').StringsVar(&f.identityF)
	f.cmd.Flag("with", "Combined classes and facts filter").Short('W').PlaceHolder("FILTER").StringsVar(&f.combinedF)
	f.cmd.Flag("dm", "Sets a discovery method (broadcast, choria)").EnumVar(&f.discoveryMethod, "broadcast", "choria", "mc")
	f.cmd.Flag("target", "Target a specific sub collective").Short('T').StringVar(&f.collective)
	f.cmd.Flag("discovery-timeout", "Timeout for doing discovery").PlaceHolder("SECONDS").IntVar(&f.discoveryTimeout)
	f.cmd.Flag("verbose", "Log verbosely").Default("false").Short('v').BoolVar(&f.verbose)
	f.cmd.Flag("json", "Produce JSON output").Short('j').BoolVar(&f.jsonFormat)

	return nil
}

func (f *findCommand) Configure() error {
	return commonConfigure()
}

func (f *findCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if f.collective == "" {
		f.collective = cfg.MainCollective
	}

	if f.discoveryTimeout == 0 {
		f.discoveryTimeout = cfg.DiscoveryTimeout
	}

	if f.discoveryMethod == "" {
		f.discoveryMethod = cfg.DefaultDiscoveryMethod
	}

	filter, err := filter.NewFilter(
		filter.FactFilter(f.factF...),
		filter.AgentFilter(f.agentsF...),
		filter.ClassFilter(f.classF...),
		filter.IdentityFilter(f.identityF...),
		filter.CombinedFilter(f.combinedF...),
	)
	if err != nil {
		return err
	}

	if f.verbose && !f.jsonFormat {
		fmt.Printf("Discovering nodes using the %s method ....\n\n", f.discoveryMethod)
	}

	var nodes []string

	start := time.Now()
	switch f.discoveryMethod {
	case "mc", "broadcast":
		nodes, err = broadcast.New(c).Discover(ctx, broadcast.Filter(filter), broadcast.Collective(f.collective), broadcast.Timeout(time.Second*time.Duration(f.discoveryTimeout)))
	case "choria":
		nodes, err = puppetdb.New(c).Discover(ctx, puppetdb.Filter(filter), puppetdb.Collective(f.collective), puppetdb.Timeout(time.Second*time.Duration(f.discoveryTimeout)))
	}
	dt := time.Since(start)

	if f.jsonFormat {
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

	if f.verbose {
		fmt.Println()
		fmt.Printf("Discovered %d nodes using the %s method in %.02f seconds\n", len(nodes), f.discoveryMethod, dt.Seconds())
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &findCommand{})
}
