package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/pretty"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/filter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	rpc "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/rpcutil"
)

type inventoryCommand struct {
	command

	ident          string
	listCollective bool
	factF          []string
	agentsF        []string
	classF         []string
	identityF      []string
	combinedF      []string
}

func (i *inventoryCommand) Setup() (err error) {
	i.cmd = cli.app.Command("inventory", "General reporting tool for nodes, collectives and subcollectives")
	i.cmd.Arg("identity", "Identity of a Choria Server to retrieve inventory from").StringVar(&i.ident)
	i.cmd.Flag("collectives", "List all known collectives").BoolVar(&i.listCollective)
	i.cmd.Flag("wf", "Match hosts with a certain fact").Short('F').StringsVar(&i.factF)
	i.cmd.Flag("wc", "Match hosts with a certain configuration management class").Short('C').StringsVar(&i.classF)
	i.cmd.Flag("wa", "Match hosts with a certain Choria agent").Short('A').StringsVar(&i.agentsF)
	i.cmd.Flag("wi", "Match hosts with a certain Choria identity").Short('I').StringsVar(&i.identityF)
	i.cmd.Flag("with", "Combined classes and facts filter").Short('W').PlaceHolder("FILTER").StringsVar(&i.combinedF)

	return
}

func (i *inventoryCommand) inventoryAgent() error {
	opts := []rpc.RequestOption{}
	if i.ident != "" {
		opts = append(opts, rpc.Targets([]string{i.ident}))
	}

	// TODO: embed rpcutil ddl somewhere and use it here or in rpc.New()
	agent, err := rpc.New(c, "rpcutil")
	if err != nil {
		return err
	}

	inventory := &rpcutil.InventoryReply{}
	stats := &rpcutil.DaemonStatsReply{}

	_, err = agent.Do(ctx, "inventory", nil, append(opts, rpc.ReplyHandler(func(_ protocol.Reply, reply *rpc.RPCReply) {
		if reply.Statuscode == mcorpc.OK {
			err = json.Unmarshal(reply.Data, inventory)
			if err != nil {
				log.Errorf("%q", reply.Data)
				log.Errorf("Could not parse inventory reply: %s", err)
			}
		}
	}))...)
	if err != nil {
		return err
	}
	if inventory == nil {
		return fmt.Errorf("no inventory received")
	}

	_, err = agent.Do(ctx, "daemon_stats", nil, append(opts, rpc.ReplyHandler(func(_ protocol.Reply, reply *rpc.RPCReply) {
		if reply.Statuscode == mcorpc.OK {
			err = json.Unmarshal(reply.Data, stats)
			if err != nil {
				log.Errorf("Could not parse daemon_stats reply: %s", err)
			}
		}
	}))...)
	if err != nil {
		return err
	}
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
	choria.SliceGroups(inventory.Agents, 3, func(g []string) {
		fmt.Printf("    %-15s %-15s %-15s\n", g[0], g[1], g[2])
	})
	fmt.Println()
	fmt.Printf("  Configuration Management Classes:\n\n")
	longest := i.longestString(inventory.Classes, 40)
	format := fmt.Sprintf("    %%-%ds %%-%ds\n", longest, longest)
	choria.SliceGroups(inventory.Classes, 2, func(g []string) {
		fmt.Printf(format, g[0], g[1])
	})
	fmt.Println()
	if len(inventory.Machines) > 0 {
		fmt.Printf("  Autonomous Agents:\n\n")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeaderLine(false)
		table.SetBorders(tablewriter.Border{Left: false, Right: false, Top: false, Bottom: false})
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		for _, m := range inventory.Machines {
			table.Append([]string{"", m.Name, m.Version, m.State})
		}
		table.Render()
		fmt.Println()
	}
	fmt.Printf("  Facts:\n\n")
	fmt.Printf("%s\n", string(pretty.PrettyOptions(inventory.Facts, &pretty.Options{
		Prefix:   "    ",
		Indent:   "  ",
		SortKeys: true,
	})))

	return nil
}

func (i *inventoryCommand) longestString(list []string, max int) int {
	longest := 0
	for _, i := range list {
		if len(i) > longest {
			longest = len(i)
		}

		if longest > max {
			return max
		}
	}

	return longest
}

func (i *inventoryCommand) inventoryCollectives() error {
	filter, err := i.parseFilterOptions()
	if err != nil {
		return err
	}

	// TODO: embed rpcutil ddl somewhere and use it here or in rpc.New()
	agent, err := rpc.New(c, "rpcutil")
	if err != nil {
		return err
	}

	collectives := make(map[string]int)
	mu := sync.Mutex{}
	res, err := agent.Do(ctx, "inventory", nil, rpc.Filter(filter), rpc.ReplyHandler(func(pr protocol.Reply, reply *rpc.RPCReply) {
		if reply.Statuscode != mcorpc.OK {
			log.Errorf("Received a error response from %s: %s", pr.SenderID(), reply.Statusmsg)
			return
		}

		inventory := &rpcutil.InventoryReply{}
		err = json.Unmarshal(reply.Data, inventory)
		if err != nil {
			log.Errorf("Could not parse inventory reply from %s: %s", pr.SenderID(), err)
		}

		mu.Lock()
		defer mu.Unlock()

		for _, c := range inventory.Collectives {
			_, ok := collectives[c]
			if !ok {
				collectives[c] = 0
			}
			collectives[c]++
		}
	}))
	if err != nil {
		return err
	}
	if res.Stats().ResponsesCount() == 0 {
		return fmt.Errorf("no responses received")
	}

	fmt.Printf("Subcollective report for %d nodes\n\n", res.Stats().OKCount())
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

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Collective", "Nodes"})
	table.SetBorders(tablewriter.Border{
		Left:   false,
		Right:  false,
		Top:    false,
		Bottom: false,
	})
	for _, kv := range cs {
		table.Append([]string{kv.Key, fmt.Sprintf("%d", kv.Value)})
	}
	table.Render()

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

func (i *inventoryCommand) parseFilterOptions() (*protocol.Filter, error) {
	return filter.NewFilter(
		filter.FactFilter(i.factF...),
		filter.AgentFilter(i.agentsF...),
		filter.ClassFilter(i.classF...),
		filter.IdentityFilter(i.identityF...),
		filter.CombinedFilter(i.combinedF...),
		filter.AgentFilter("rpcutil"),
	)
}

func (i *inventoryCommand) Configure() error {
	protocol.ClientStrictValidation = false

	return commonConfigure()
}

func init() {
	cli.commands = append(cli.commands, &inventoryCommand{})
}
