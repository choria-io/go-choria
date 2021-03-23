package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/client/rpcutilclient"
	"github.com/choria-io/go-choria/internal/util"
)

type factsCommand struct {
	fact    string
	verbose bool
	json    bool
	table   bool
	nodes   bool
	reverse bool

	fo *discovery.StandardOptions

	command
}

type factCommandValue struct {
	value string
	Cnt   int      `json:"count"`
	Nodes []string `json:"identities"`
}

func (f *factsCommand) Setup() error {
	f.cmd = cli.app.Command("facts", "Reports on usage for a specific fact")
	f.cmd.Arg("fact", "The fact to report on").Required().StringVar(&f.fact)
	f.cmd.Flag("table", "Produce tabular output").Short('t').BoolVar(&f.table)
	f.cmd.Flag("json", "Produce JSON output").Default("false").Short('j').BoolVar(&f.json)
	f.cmd.Flag("verbose", "Log verbosely").Default("false").Short('v').BoolVar(&f.verbose)
	f.cmd.Flag("show-nodes", "Show matching nodes").Default("false").Short('n').BoolVar(&f.nodes)
	f.cmd.Flag("reverse", "Reverse sorting order").Short('r').BoolVar(&f.reverse)

	f.fo = discovery.NewStandardOptions()
	f.fo.AddFilterFlags(f.cmd)
	f.fo.AddSelectionFlags(f.cmd)
	f.fo.AddFlatFileFlags(f.cmd)

	return nil
}

func (f *factsCommand) Configure() error {
	return commonConfigure()
}

func (f *factsCommand) showJson(facts interface{}) error {
	j, err := json.MarshalIndent(facts, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(j))

	return nil
}

func (f *factsCommand) sortByCount(facts map[string]*factCommandValue) []*factCommandValue {
	res := []*factCommandValue{}
	for _, v := range facts {
		res = append(res, v)
	}

	sort.Slice(res, func(i, j int) bool {
		if f.reverse {
			return res[i].Cnt > res[j].Cnt
		}

		return res[i].Cnt < res[j].Cnt
	})

	return res
}

func (f *factsCommand) showTable(facts map[string]*factCommandValue) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(true)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	if f.verbose || f.nodes {
		table.SetHeader([]string{"Fact", "Matches", "Nodes"})
	} else {
		table.SetHeader([]string{"Fact", "Matches"})
	}

	for _, v := range f.sortByCount(facts) {
		if f.verbose || f.nodes {
			sort.Strings(v.Nodes)
			table.Append([]string{v.value, strconv.Itoa(v.Cnt), strings.Join(v.Nodes, "\n")})
		} else {
			table.Append([]string{v.value, strconv.Itoa(v.Cnt)})
		}
	}

	table.Render()

	return nil
}

func (f *factsCommand) showText(res *rpcutilclient.GetFactResult, facts map[string]*factCommandValue, logger *logrus.Entry) error {
	fmt.Printf("Report for fact: %s\n\n", f.fact)

	vals := []string{}
	for k := range facts {
		vals = append(vals, k)
	}
	longest := util.LongestString(vals, 4000)
	format := fmt.Sprintf("  %%-%ds found %%d times\n", longest)

	for _, v := range f.sortByCount(facts) {
		fmt.Printf(format, v.value, v.Cnt)
		if f.verbose || f.nodes {
			fmt.Println()
			sort.Strings(v.Nodes)
			for _, n := range v.Nodes {
				fmt.Printf("    %s\n", n)
			}
			fmt.Println()
		}
	}

	fmt.Println()

	res.RenderResults(os.Stdout, rpcutilclient.TXTFooter, rpcutilclient.DisplayAll, f.verbose, false, cfg.Color, logger)

	return nil
}

func (f *factsCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	logger := c.Logger("facts")
	f.fo.SetDefaultsFromChoria(c)

	start := time.Now()
	nodes, _, err := f.fo.Discover(ctx, c, "rpcutil", true, false, c.Logger("facts"))
	if err != nil {
		return err
	}
	finish := time.Now()

	if len(nodes) == 0 {
		return fmt.Errorf("did not discover any nodes")
	}

	c, err := rpcutilclient.New(rpcutilclient.Logger(logger))
	if err != nil {
		return err
	}

	c.OptionTargets(nodes)

	res, err := c.GetFact(f.fact).Do(ctx)
	if err != nil {
		return err
	}

	if res.Stats().OKCount() == 0 {
		return fmt.Errorf("no responses received")
	}

	res.Stats().OverrideDiscoveryTime(start, finish)

	facts := map[string]*factCommandValue{}

	res.EachOutput(func(o *rpcutilclient.GetFactOutput) {
		if !o.ResultDetails().OK() {
			logger.Errorf("received an error from %s: %s", o.ResultDetails().Sender(), o.ResultDetails().StatusMessage())
			return
		}

		var ok bool
		vjs := "nil"

		if o.Value() != nil {
			vjs, ok = o.Value().(string)
			if !ok {
				vj, err := json.Marshal(o.Value())
				if err != nil {
					logger.Errorf("could not process result from %s: %s", o.ResultDetails().Sender(), err)
				}

				vjs = string(vj)
			}
		}

		_, ok = facts[vjs]
		if !ok {
			facts[vjs] = &factCommandValue{
				value: vjs,
				Cnt:   0,
				Nodes: []string{},
			}
		}

		facts[vjs].Cnt++
		facts[vjs].Nodes = append(facts[vjs].Nodes, o.ResultDetails().Sender())
	})

	if len(facts) == 0 {
		return fmt.Errorf("no facts returned")
	}

	switch {
	case f.json:
		return f.showJson(facts)
	case f.table:
		return f.showTable(facts)
	default:
		return f.showText(res, facts, logger)
	}
}

func init() {
	cli.commands = append(cli.commands, &factsCommand{})
}
