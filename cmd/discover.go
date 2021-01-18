package cmd

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/client/discovery"
)

type discoverCommand struct {
	command

	jsonFormat bool
	verbose    bool

	fo *discovery.StandardOptions
}

func (d *discoverCommand) Setup() error {
	d.cmd = cli.app.Command("discover", "Discover nodes using the discovery system matching filter criteria").Alias("find")

	d.fo = discovery.NewStandardOptions()
	d.fo.AddFilterFlags(d.cmd)
	d.fo.AddSelectionFlags(d.cmd)
	d.fo.AddFlatFileFlags(d.cmd)

	d.cmd.Flag("verbose", "Log verbosely").Default("false").Short('v').BoolVar(&d.verbose)
	d.cmd.Flag("json", "Produce JSON output").Short('j').BoolVar(&d.jsonFormat)

	return nil
}

func (d *discoverCommand) Configure() error {
	return commonConfigure()
}

func (d *discoverCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	d.fo.SetDefaults(cfg.MainCollective, cfg.DefaultDiscoveryMethod, cfg.DiscoveryTimeout)

	nodes, dt, err := d.fo.Discover(ctx, c, "", true, d.verbose && !d.jsonFormat, c.Logger("discovery"))
	if err != nil {
		return err
	}

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
		fmt.Printf("Discovered %d nodes using the %s method in %.02f seconds\n", len(nodes), d.fo.DiscoveryMethod, dt.Seconds())
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &discoverCommand{})
}
