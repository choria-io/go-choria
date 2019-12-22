package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-client/discovery/broadcast"
	"github.com/choria-io/go-protocol/filter"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	rpc "github.com/choria-io/mcorpc-agent-provider/mcorpc/client"
	agentddl "github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"

	"github.com/fatih/color"
	"github.com/gosuri/uiprogress"
)

type reqCommand struct {
	command

	agent           string
	action          string
	args            map[string]string
	ddl             *agentddl.DDL
	actionInterface *agentddl.Action

	discoveryTimeout int
	displayOverride  string
	noProgress       bool
	discoveryTime    time.Duration
	startTime        time.Time
	limit            string
	limitSeed        int64
	batch            int
	batchSleep       int
	verbose          bool
	jsonOnly         bool
	silent           bool
	nodesFile        string
	workers          int
	collective       string
	factF            []string
	agentsF          []string
	classF           []string
	identityF        []string
	combinedF        []string
	outputFile       string

	outputWriter     *bufio.Writer
	outputFileHandle *os.File
}

type rpcStats struct {
	RequestID           string   `json:"requestid"`
	NoResponses         []string `json:"no_responses"`
	UnexpectedResponses []string `json:"unexpected_responses"`
	DiscoveredCount     int      `json:"discovered"`
	FailCount           int      `json:"failed"`
	OKCount             int      `json:"ok"`
	ResponseCount       int      `json:"responses"`
	PublishTime         float32  `json:"publish_time"`
	RequestTime         float32  `json:"request_time"`
	DiscoverTime        float32  `json:"discover_time"`
	StartTime           int64    `json:"start_time_utc"`
}

type rpcReply struct {
	Sender string `json:"sender"`
	*rpc.RPCReply
}

type rpcResults struct {
	Agent     string          `json:"agent"`
	Action    string          `json:"action"`
	Replies   []*rpcReply     `json:"replies"`
	Stats     *rpcStats       `json:"request_stats"`
	Summaries json.RawMessage `json:"summaries"`
}

func (r *reqCommand) Setup() (err error) {
	r.args = make(map[string]string)

	r.cmd = cli.app.Command("req", "Performs a RPC request against the Choria network")
	r.cmd.Alias("rpc")

	r.cmd.Arg("agent", "The agent to invoke").Required().StringVar(&r.agent)
	r.cmd.Arg("action", "The action to invoke").Required().StringVar(&r.action)
	r.cmd.Arg("args", "Arguments to pass to the action in key=val format").StringMapVar(&r.args)

	r.cmd.Flag("wf", "Match hosts with a certain fact").Short('F').StringsVar(&r.factF)
	r.cmd.Flag("wc", "Match hosts with a certain configuration management class").Short('C').StringsVar(&r.classF)
	r.cmd.Flag("wa", "Match hosts with a certain Choria agent").Short('A').StringsVar(&r.agentsF)
	r.cmd.Flag("wi", "Match hosts with a certain Choria identity").Short('I').StringsVar(&r.identityF)
	r.cmd.Flag("with", "Combined classes and facts filter").Short('W').PlaceHolder("FILTER").StringsVar(&r.combinedF)
	r.cmd.Flag("limit", "Limits request to a set of targets eg 10 or 10%").StringVar(&r.limit)
	r.cmd.Flag("limit-seed", "Seed value for deterministic random limits").PlaceHolder("SEED").Int64Var(&r.limitSeed)
	r.cmd.Flag("batch", "Do requests in batches").PlaceHolder("SIZE").IntVar(&r.batch)
	r.cmd.Flag("batch-sleep", "Sleep time between batches").PlaceHolder("SECONDS").IntVar(&r.batchSleep)
	r.cmd.Flag("target", "Target a specific sub collective").Short('T').StringVar(&r.collective)
	r.cmd.Flag("workers", "How many workers to start for receiving messages").Default("3").IntVar(&r.workers)
	r.cmd.Flag("nodes", "List of nodes to interact with").ExistingFileVar(&r.nodesFile)
	r.cmd.Flag("np", "Disable the progress bar").BoolVar(&r.noProgress)
	r.cmd.Flag("json", "Produce JSON output only").Short('j').BoolVar(&r.jsonOnly)
	r.cmd.Flag("verbose", "Enable verbose output").Short('v').BoolVar(&r.verbose)
	r.cmd.Flag("display", "Display only a subset of results (ok, failed, all, none)").EnumVar(&r.displayOverride, "ok", "failed", "all", "none")
	r.cmd.Flag("discovery-timeout", "Timeout for doing discovery").PlaceHolder("SECONDS").IntVar(&r.discoveryTimeout)
	r.cmd.Flag("output-file", "Filename to write output to").PlaceHolder("FILENAME").Short('o').StringVar(&r.outputFile)

	return
}

func (r *reqCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	r.startTime = time.Now()

	if r.outputFile != "" {
		r.outputFileHandle, err = os.Create(r.outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output-file: %s", err)
		}
	} else {
		r.outputFileHandle = os.Stdout
	}
	r.outputWriter = bufio.NewWriter(r.outputFileHandle)

	if r.jsonOnly {
		r.silent = true
	}

	if r.collective == "" {
		r.collective = cfg.MainCollective
	}

	if r.discoveryTimeout == 0 {
		r.discoveryTimeout = cfg.DiscoveryTimeout
	}

	r.ddl, err = agentddl.Find(r.agent, cfg.LibDir)
	if err != nil {
		return fmt.Errorf("could not find DDL for agent %s: %s", r.agent, err)
	}

	r.actionInterface, err = r.ddl.ActionInterface(r.action)
	if err != nil {
		return err
	}

	input, _, err := r.actionInterface.ValidateAndConvertToDDLTypes(r.args)
	if err != nil {
		return fmt.Errorf("invalid input: %s", err)
	}

	filter, err := filter.NewFilter(
		filter.FactFilter(r.factF...),
		filter.AgentFilter(r.agentsF...),
		filter.ClassFilter(r.classF...),
		filter.IdentityFilter(r.identityF...),
		filter.CombinedFilter(r.combinedF...),
		filter.AgentFilter(r.agent),
	)
	if err != nil {
		return fmt.Errorf("could not parse filters: %s", err)
	}

	dstart := time.Now()
	nodes, err := r.discover(filter)
	if err != nil {
		return fmt.Errorf("could not discover nodes")
	}
	r.discoveryTime = time.Since(dstart)

	expected := len(nodes)
	if expected == 0 {
		return fmt.Errorf("did not discover any nodes")
	}

	if r.silent {
		r.noProgress = true
	}

	var bar *uiprogress.Bar

	progressSetup := func(count int) {
		if !r.noProgress {
			bar = uiprogress.AddBar(count).AppendCompleted().PrependElapsed()
			fmt.Println()

			bar.PrependFunc(func(b *uiprogress.Bar) string {
				if b.Current() < expected {
					return color.RedString(fmt.Sprintf("%d / %d", b.Current(), count))
				}

				return color.GreenString(fmt.Sprintf("%d / %d", b.Current(), count))
			})

			uiprogress.Start()
		}
	}

	results := rpcResults{
		Agent:   r.agent,
		Action:  r.action,
		Replies: []*rpcReply{},
	}

	handler := func(pr protocol.Reply, reply *rpc.RPCReply) {
		if !r.noProgress {
			bar.Incr()
		}

		results.Replies = append(results.Replies, &rpcReply{pr.SenderID(), reply})
	}

	opts := []rpc.RequestOption{
		rpc.Collective(r.collective),
		rpc.Targets(nodes),
		rpc.ReplyHandler(handler),
		rpc.Workers(r.workers),
		rpc.LimitMethod(cfg.RPCLimitMethod),
		rpc.DiscoveryEndCB(func(d, l int) error {
			progressSetup(l)
			return nil
		}),
	}

	if r.batch > 0 {
		if r.batchSleep == 0 {
			r.batchSleep = 1
		}

		opts = append(opts, rpc.InBatches(r.batch, r.batchSleep))
	}

	if r.limit != "" {
		opts = append(opts, rpc.LimitSize(r.limit))
	}

	if r.limitSeed > 0 {
		opts = append(opts, rpc.LimitSeed(r.limitSeed))
	}

	agent, err := rpc.New(c, r.agent, rpc.DDL(r.ddl))
	if err != nil {
		return fmt.Errorf("could not create client: %s", err)
	}

	rpcres, err := agent.Do(ctx, r.action, input, opts...)
	if err != nil {
		return fmt.Errorf("could not perform request: %s", err)
	}

	results.Stats, err = r.statsFromClient(rpcres.Stats())
	if err != nil {
		return fmt.Errorf("could not process stats: %s", err)
	}

	if !r.noProgress {
		uiprogress.Stop()
		fmt.Println()
	}

	err = r.displayResults(&results)
	if err != nil {
		return fmt.Errorf("could not display results: %s", err)
	}

	return
}

func (r *reqCommand) displayResults(res *rpcResults) error {
	if r.jsonOnly {
		return r.displayResultsAsJSON(res)
	}

	return r.displayResultsAsTXT(res)
}

func (r *reqCommand) displayResultsAsTXT(res *rpcResults) error {
	status := map[mcorpc.StatusCode]string{
		mcorpc.OK:            "",
		mcorpc.Aborted:       color.RedString("Request Aborted"),
		mcorpc.InvalidData:   color.YellowString("Invalid Request Data"),
		mcorpc.MissingData:   color.YellowString("Missing Request Data"),
		mcorpc.UnknownAction: color.YellowString("Unknown Action"),
		mcorpc.UnknownError:  color.RedString("Unknown Request Status"),
	}

	for _, reply := range res.Replies {
		show := false

		if r.displayOverride == "" {
			if reply.Statuscode > mcorpc.OK && r.actionInterface.Display == "failed" {
				show = true
			} else if reply.Statuscode > mcorpc.OK && r.actionInterface.Display == "" {
				show = true
			} else if r.actionInterface.Display == "ok" && reply.Statuscode == mcorpc.OK {
				show = true
			} else if r.actionInterface.Display == "always" {
				show = true
			}
		} else if r.displayOverride == "ok" {
			if reply.Statuscode == mcorpc.OK {
				show = true
			}
		} else if r.displayOverride == "failed" {
			if reply.Statuscode > mcorpc.OK {
				show = true
			}
		} else if r.displayOverride == "all" {
			show = true
		} else if r.displayOverride == "none" {
			show = false
		}

		basicPrinter := func(data json.RawMessage) {
			if !show {
				return
			}

			j, err := json.MarshalIndent(data, "   ", "   ")
			if err != nil {
				fmt.Fprintf(r.outputWriter, "   %s\n", string(data))
			}

			fmt.Fprintf(r.outputWriter, "   %s\n", string(j))

			r.outputWriter.Flush()
		}

		errorPrinter := func(m string) {
			fmt.Fprintf(r.outputWriter, "    %s\n", color.YellowString(m))

			r.outputWriter.Flush()
		}

		ddlAssistedPrinter := func(data map[string]interface{}, raw []byte) {
			max := 0
			keys := []string{}

			for key := range data {
				output, ok := r.actionInterface.Output[key]
				if ok {
					if len(output.DisplayAs) > max {
						max = len(output.DisplayAs)
					}
				} else {
					if len(key) > max {
						max = len(key)
					}
				}

				keys = append(keys, key)
			}

			formatStr := fmt.Sprintf("%%%ds: %%s\n", max+3)
			prefixFormatStr := fmt.Sprintf("%%%ds", max+5)

			sort.Strings(keys)

			for _, key := range keys {
				val := gjson.GetBytes(raw, key)
				keyStr := key
				valStr := val.String()

				output, ok := r.actionInterface.Output[key]
				if ok {
					keyStr = output.DisplayAs
				}

				if val.IsArray() || val.IsObject() {
					valStr = string(pretty.PrettyOptions([]byte(valStr), &pretty.Options{
						SortKeys: true,
						Prefix:   fmt.Sprintf(prefixFormatStr, " "),
						Indent:   "   ",
						Width:    80,
					}))
				}

				fmt.Fprintf(r.outputWriter, formatStr, keyStr, strings.TrimLeft(valStr, " "))
			}

			r.outputWriter.Flush()
		}

		parsed, ok := gjson.ParseBytes(reply.RPCReply.Data).Value().(map[string]interface{})
		if ok {
			r.actionInterface.SetOutputDefaults(parsed)
			r.actionInterface.AggregateResult(parsed)
		}

		if show {
			fmt.Fprintf(r.outputWriter, "%-40s %s\n", reply.Sender, status[reply.Statuscode])

			if r.verbose {
				basicPrinter(reply.RPCReply.Data)
			} else {
				if reply.RPCReply.Statuscode > mcorpc.OK {
					errorPrinter(reply.RPCReply.Statusmsg)
				} else {
					ddlAssistedPrinter(parsed, reply.RPCReply.Data)
				}

				fmt.Fprintln(r.outputWriter)
			}
		}
	}

	if r.silent {
		return nil
	}

	summaryPrinter := func(summaries map[string][]string) {
		keys := []string{}
		for k := range summaries {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			descr := k
			output, ok := r.actionInterface.Output[k]
			if ok {
				descr = output.DisplayAs
			}

			fmt.Fprintln(r.outputWriter, color.HiWhiteString("Summary of %s:\n", descr))
			if len(summaries[k]) == 0 {
				fmt.Fprintf(r.outputWriter, "   %s\n\n", color.YellowString("No summary received"))
				continue
			}

			for _, v := range summaries[k] {
				if strings.ContainsRune(v, '\n') {
					fmt.Fprintln(r.outputWriter, v)
				} else {
					fmt.Fprintf(r.outputWriter, "   %s\n", v)
				}

			}
			fmt.Fprintln(r.outputWriter)
		}
	}

	summaries, err := r.actionInterface.AggregateSummaryFormattedStrings()
	if err == nil {
		summaryPrinter(summaries)
	}

	fmt.Fprintln(r.outputWriter)

	if r.verbose {
		fmt.Fprintln(r.outputWriter, color.YellowString("---- request stats ----"))
		fmt.Fprintf(r.outputWriter, "               Nodes: %d / %d\n", res.Stats.ResponseCount, res.Stats.DiscoveredCount)
		fmt.Fprintf(r.outputWriter, "         Pass / Fail: %d / %d\n", res.Stats.OKCount, res.Stats.FailCount)
		fmt.Fprintf(r.outputWriter, "        No Responses: %d\n", len(res.Stats.NoResponses))
		fmt.Fprintf(r.outputWriter, "Unexpected Responses: %d\n", len(res.Stats.UnexpectedResponses))
		fmt.Fprintf(r.outputWriter, "          Start Time: %s\n", time.Unix(res.Stats.StartTime, 0).Format("2006-01-02T15:04:05-0700"))
		fmt.Fprintf(r.outputWriter, "      Discovery Time: %v\n", time.Duration(res.Stats.DiscoverTime*1000000000))
		fmt.Fprintf(r.outputWriter, "        Publish Time: %v\n", time.Duration(res.Stats.PublishTime*1000000000))
		fmt.Fprintf(r.outputWriter, "          Agent Time: %v\n", time.Duration((res.Stats.RequestTime-res.Stats.PublishTime)*1000000000))
		fmt.Fprintf(r.outputWriter, "          Total Time: %v\n", time.Duration((res.Stats.RequestTime+res.Stats.DiscoverTime)*1000000000))
	} else {
		fmt.Fprintf(r.outputWriter, "Finished processing %d / %d hosts in %s\n", res.Stats.ResponseCount, res.Stats.DiscoveredCount, time.Duration((res.Stats.RequestTime)*1000000000))
	}

	nodeListPrinter := func(nodes []string, message string) {
		if len(nodes) > 0 {
			fmt.Fprintf(r.outputWriter, "\n%s: %d\n\n", message, len(nodes))

			w := new(tabwriter.Writer)
			w.Init(r.outputFileHandle, 0, 0, 4, ' ', 0)

			choria.SliceGroups(nodes, 3, func(g []string) {
				fmt.Fprintln(w, "    "+strings.Join(g, "\t")+"\t")
			})

			w.Flush()
		}
	}

	nodeListPrinter(res.Stats.NoResponses, "No Responses from")
	nodeListPrinter(res.Stats.UnexpectedResponses, "Unexpected Responses from")

	r.outputWriter.Flush()

	return nil
}

func (r *reqCommand) displayResultsAsJSON(res *rpcResults) error {
	var err error

	for _, reply := range res.Replies {
		parsed, ok := gjson.ParseBytes(reply.RPCReply.Data).Value().(map[string]interface{})
		if ok {
			r.actionInterface.SetOutputDefaults(parsed)
			r.actionInterface.AggregateResult(parsed)
		}
	}

	// silently failing as this is optional
	res.Summaries, _ = r.actionInterface.AggregateSummaryJSON()

	j, err := json.MarshalIndent(res, "", "   ")
	if err != nil {
		return fmt.Errorf("could not prepare display: %s", err)
	}

	fmt.Fprintln(r.outputWriter, string(j))

	r.outputWriter.Flush()

	return nil
}

func (r *reqCommand) Configure() error {
	protocol.ClientStrictValidation = false

	return commonConfigure()
}

func (r *reqCommand) discover(filter *protocol.Filter) ([]string, error) {
	if r.nodesFile != "" {
		file, err := os.Open(r.nodesFile)
		if err != nil {
			return []string{}, err
		}
		defer file.Close()

		found := []string{}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			found = append(found, strings.TrimSpace(scanner.Text()))
		}

		err = scanner.Err()
		if err != nil {
			return []string{}, err
		}

		return found, nil
	}

	if !r.silent {
		fmt.Print("Discovering nodes .... ")
	}

	nodes, err := broadcast.New(c).Discover(ctx, broadcast.Filter(filter), broadcast.Timeout(time.Second*time.Duration(r.discoveryTimeout)))

	if !r.silent {
		fmt.Printf("%d\n", len(nodes))
	}

	return nodes, err
}

func (r *reqCommand) statsFromClient(cs *rpc.Stats) (*rpcStats, error) {
	s := &rpcStats{}

	s.RequestID = cs.RequestID
	s.NoResponses = cs.NoResponseFrom()
	s.UnexpectedResponses = cs.UnexpectedResponseFrom()
	s.DiscoveredCount = cs.DiscoveredCount()
	s.FailCount = cs.FailCount()
	s.OKCount = cs.OKCount()
	s.ResponseCount = cs.ResponsesCount()
	s.StartTime = r.startTime.UTC().Unix()
	s.DiscoverTime = float32(r.discoveryTime) / 1000000000

	d, err := cs.PublishDuration()
	if err == nil {
		s.PublishTime = float32(d) / 1000000000
	}

	d, err = cs.RequestDuration()
	if err == nil {
		s.RequestTime = float32(d) / 1000000000
	}

	return s, nil
}

func init() {
	cli.commands = append(cli.commands, &reqCommand{})
}
