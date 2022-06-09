// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/choria-io/go-choria/internal/fs"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/discovery"
)

type discoverCommand struct {
	command

	jsonFormat bool
	verbose    bool
	silent     bool
	fo         *discovery.StandardOptions
}

func (d *discoverCommand) Setup() error {
	d.cmd = cli.app.Command("discover", "Discover nodes using the discovery system matching filter criteria").Alias("find")
	d.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)
	d.cmd.CheatFile(fs.FS, "discover", "cheats/discover.md")

	d.fo = discovery.NewStandardOptions()
	d.fo.AddFilterFlags(d.cmd)
	d.fo.AddSelectionFlags(d.cmd)
	d.fo.AddFlatFileFlags(d.cmd)

	d.cmd.Flag("verbose", "Log verbosely").Default("false").Short('v').BoolVar(&d.verbose)
	d.cmd.Flag("json", "Produce JSON output").Short('j').BoolVar(&d.jsonFormat)
	d.cmd.Flag("silent", "Produce as little logging as possible").Hidden().BoolVar(&d.silent)

	return nil
}

func (d *discoverCommand) Configure() error {
	err = commonConfigure()
	if err != nil {
		return err
	}

	if d.silent {
		logrus.SetOutput(os.Stderr)
		logrus.SetLevel(logrus.PanicLevel)
	}

	cfg.LogLevel = "fatal"

	return nil
}

func (d *discoverCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	err = d.run()
	if err != nil {
		if d.silent {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	}

	return err
}

func (d *discoverCommand) run() (err error) {
	d.fo.SetDefaultsFromChoria(c)
	nodes, dt, err := d.fo.Discover(ctx, c, "rpcutil", true, d.verbose && !d.jsonFormat, c.Logger("discovery"))
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
