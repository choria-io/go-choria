package cmd

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/client/discovery/broadcast"
	"github.com/choria-io/go-choria/client/discovery/puppetdb"
)

type discoverCommand struct {
	command

	jsonFormat bool
	verbose    bool

	fo stdFilterOptions
}

func (d *discoverCommand) Setup() error {
	d.cmd = cli.app.Command("discover", "Discover nodes using the discovery system matching filter criteria").Alias("find")

	addStdFilter(d.cmd, &d.fo)
	addStdDiscovery(d.cmd, &d.fo)

	d.cmd.Flag("verbose", "Log verbosely").Default("false").Short('v').BoolVar(&d.verbose)
	d.cmd.Flag("json", "Produce JSON output").Short('j').BoolVar(&d.jsonFormat)

	return nil
}

func (d *discoverCommand) Configure() error {
	return commonConfigure()
}

func (d *discoverCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	d.fo.setDefaults(cfg.MainCollective, cfg.DefaultDiscoveryMethod, cfg.DiscoveryTimeout)

	filter, err := d.fo.newFilter("")
	if err != nil {
		return err
	}

	if d.verbose && !d.jsonFormat {
		fmt.Printf("Discovering nodes using the %s method ....\n\n", d.fo.dm)
	}

	var nodes []string

	start := time.Now()
	switch d.fo.dm {
	case "mc", "broadcast":
		nodes, err = broadcast.New(c).Discover(ctx, broadcast.Filter(filter), broadcast.Collective(d.fo.collective), broadcast.Timeout(time.Second*time.Duration(d.fo.dt)))
	case "choria":
		nodes, err = puppetdb.New(c).Discover(ctx, puppetdb.Filter(filter), puppetdb.Collective(d.fo.collective), puppetdb.Timeout(time.Second*time.Duration(d.fo.dt)))
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
		fmt.Printf("Discovered %d nodes using the %s method in %.02f seconds\n", len(nodes), d.fo.dm, dt.Seconds())
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &discoverCommand{})
}
