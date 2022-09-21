// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/pretty"

	"github.com/choria-io/go-choria/client/rpcutilclient"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/rpcutil"
)

type inventoryCommand struct {
	command

	ident          string
	listCollective bool
	showFacts      bool

	factF     []string
	agentsF   []string
	classF    []string
	identityF []string
	combinedF []string
}

func (i *inventoryCommand) Setup() (err error) {
	i.cmd = cli.app.Command("inventory", "Reporting tool for nodes, collectives and sub-collectives")
	i.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)
	i.cmd.Arg("identity", "Identity of a Choria Server to retrieve inventory from").StringVar(&i.ident)
	i.cmd.Flag("collectives", "List all known collectives").UnNegatableBoolVar(&i.listCollective)
	i.cmd.Flag("facts", "Enable or disable displaying of facts").Default("true").BoolVar(&i.showFacts)
	i.cmd.Flag("wf", "Match hosts with a certain fact").Short('F').StringsVar(&i.factF)
	i.cmd.Flag("wc", "Match hosts with a certain tagged class").Short('C').StringsVar(&i.classF)
	i.cmd.Flag("wa", "Match hosts with a certain Choria agent").Short('A').StringsVar(&i.agentsF)
	i.cmd.Flag("wi", "Match hosts with a certain Choria identity").Short('I').StringsVar(&i.identityF)
	i.cmd.Flag("with", "Combined classes and facts filter").Short('W').PlaceHolder("FILTER").StringsVar(&i.combinedF)

	return
}

func (i *inventoryCommand) inventoryAgent() error {
	if i.ident == "" {
		return fmt.Errorf("identity is required")
	}

	rpcu, err := rpcutilclient.New(c, rpcutilclient.Logger(c.Logger("inventory")))
	if err != nil {
		return err
	}
	rpcu.OptionTargets([]string{i.ident})
	rpcu.OptionWorkers(1)

	inventory := &rpcutil.InventoryReply{}
	stats := &rpcutil.DaemonStatsReply{}

	ires, err := rpcu.Inventory().Do(ctx)
	if err != nil {
		return err
	}
	ires.EachOutput(func(r *rpcutilclient.InventoryOutput) {
		if !r.ResultDetails().OK() {
			log.Errorf("inventory failed for %s: %s", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage())
			return
		}

		err = r.ParseInventoryOutput(inventory)
		if err != nil {
			log.Errorf("could not parse inventory from %s: %s", r.ResultDetails().Sender(), err)
			return
		}
	})

	if inventory.Version == "" {
		return fmt.Errorf("no inventory received")
	}

	sres, err := rpcu.DaemonStats().Do(ctx)
	if err != nil {
		return err
	}
	sres.EachOutput(func(r *rpcutilclient.DaemonStatsOutput) {
		if !r.ResultDetails().OK() {
			log.Errorf("daemon_stats failed for %s: %s", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage())
			return
		}

		err := r.ParseDaemonStatsOutput(stats)
		if err != nil {
			log.Errorf("could not parse daemon_stats from %s: %s", r.ResultDetails().Sender(), err)
			return
		}
	})
	if stats == nil {
		return fmt.Errorf("no daemon_stats received")
	}

	fmt.Printf("Inventory for %s\n\n", i.ident)
	fmt.Printf("  Choria Server Statistics:\n\n")
	fmt.Printf("                    Version: %s\n", inventory.Version)
	fmt.Printf("                 Start Time: %s\n", time.Unix(stats.StartTime, 0).Round(time.Second))
	fmt.Printf("                Config File: %s\n", stats.ConfigFile)
	fmt.Printf("                Collectives: %s\n", strings.Join(inventory.Collectives, ", "))
	fmt.Printf("            Main Collective: %s\n", inventory.MainCollective)
	fmt.Printf("                 Process ID: %d\n", stats.PID)
	fmt.Printf("             Total Messages: %d\n", stats.Total)
	fmt.Printf("    Messages Passed Filters: %d\n", stats.Passed)
	fmt.Printf("          Messages Filtered: %d\n", stats.Filtered)
	fmt.Printf("           Expired Messages: %d\n", stats.TTLExpired)
	fmt.Printf("               Replies Sent: %d\n", stats.Replies)
	fmt.Println()
	fmt.Printf("  Agents:\n\n")
	util.SliceVerticalGroups(inventory.Agents, 3, func(g []string) {
		fmt.Printf("    %-15s %-15s %-15s\n", g[0], g[1], g[2])
	})
	fmt.Println()
	fmt.Printf("  Tagged Classes:\n\n")
	longest := util.LongestString(inventory.Classes, 40)
	format := fmt.Sprintf("    %%-%ds %%-%ds\n", longest, longest)
	util.SliceVerticalGroups(inventory.Classes, 2, func(g []string) {
		fmt.Printf(format, g[0], g[1])
	})
	fmt.Println()
	if len(inventory.Machines) > 0 {
		fmt.Printf("  Autonomous Agents:\n\n")
		table := new(tabwriter.Writer)
		table.Init(os.Stdout, 0, 0, 4, ' ', 0)

		for _, m := range inventory.Machines {
			fmt.Fprintf(table, "    %s\t%s\t%s\t\n", m.Name, m.Version, m.State)
		}
		table.Flush()
		fmt.Println()
	}

	if i.showFacts {
		fmt.Printf("  Facts:\n\n")
		fmt.Printf("%s\n", string(pretty.PrettyOptions(inventory.Facts, &pretty.Options{
			Prefix:   "    ",
			Indent:   "  ",
			SortKeys: true,
		})))
	}

	return nil
}

func (i *inventoryCommand) inventoryCollectives() error {
	rpcu, err := rpcutilclient.New(c, rpcutilclient.Logger(c.Logger("inventory")))
	if err != nil {
		return err
	}

	rpcu.OptionFactFilter(i.factF...)
	rpcu.OptionClassFilter(i.classF...)
	rpcu.OptionIdentityFilter(i.identityF...)
	rpcu.OptionCombinedFilter(i.combinedF...)
	rpcu.OptionAgentFilter(i.agentsF...)

	collectives := make(map[string]int)

	res, err := rpcu.CollectiveInfo().Do(ctx)
	if err != nil {
		return err
	}

	res.EachOutput(func(reply *rpcutilclient.CollectiveInfoOutput) {
		if !reply.ResultDetails().OK() {
			log.Errorf("Received a error response from %s: %s", reply.ResultDetails().Sender(), reply.ResultDetails().StatusMessage())
			return
		}

		inventory := &rpcutil.CollectiveInfoReply{}
		err = reply.ParseCollectiveInfoOutput(&inventory)
		if err != nil {
			log.Errorf("Could not parse inventory reply from %s: %s", reply.ResultDetails().Sender(), err)
			return
		}

		for _, c := range inventory.Collectives {
			_, ok := collectives[c]
			if !ok {
				collectives[c] = 0
			}
			collectives[c]++
		}
	})

	type kv struct {
		Key   string
		Value int
	}

	var cs []kv
	for k, v := range collectives {
		cs = append(cs, kv{k, v})
	}

	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Value > cs[j].Value
	})

	table := util.NewUTF8TableWithTitle(fmt.Sprintf("Collective Report for %d nodes", res.Stats().OKCount()), "Collective", "Nodes")
	for _, kv := range cs {
		table.AddRow(kv.Key, fmt.Sprintf("%d", kv.Value))
	}

	fmt.Println(table.Render())

	if len(res.Stats().NoResponseFrom()) > 0 {
		res.RenderResults(os.Stdout, rpcutilclient.TXTFooter, rpcutilclient.DisplayAll, debug, false, c.Config.Color, c.Logger("inventory"))
	}

	return nil
}

func (i *inventoryCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	switch {
	case i.ident != "":
		return i.inventoryAgent()
	case i.listCollective:
		return i.inventoryCollectives()
	default:
		return fmt.Errorf("please specify a node to retrieve inventory for")
	}
}

func (i *inventoryCommand) Configure() error {
	protocol.ClientStrictValidation = false

	return commonConfigure()
}

func init() {
	cli.commands = append(cli.commands, &inventoryCommand{})
}
