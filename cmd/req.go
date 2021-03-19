package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gosuri/uiprogress"

	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/protocol"
	rpc "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	agentddl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/replyfmt"
)

type reqCommand struct {
	command

	agent           string
	action          string
	args            map[string]string
	ddl             *agentddl.DDL
	actionInterface *agentddl.Action
	progressBar     *uiprogress.Bar
	input           map[string]interface{}
	filter          *protocol.Filter

	displayOverride string
	noProgress      bool
	startTime       time.Time
	limit           string
	limitSeed       int64
	batch           int
	batchSleep      int
	verbose         bool
	jsonOnly        bool
	tableOnly       bool
	silent          bool
	workers         int

	fo *discovery.StandardOptions

	outputFile string
	exprFilter string

	outputWriter     *bufio.Writer
	outputFileHandle *os.File
}

func (r *reqCommand) Setup() (err error) {
	r.args = make(map[string]string)

	help := `Performs a RPC request against the Choria network

Replies are shown according to DDL rules or --display, replies can also
be filtered using an expression language that will include only those replies
that match the filter.

   # include only OK responses
   --filter-replies 'ok()'

   # include only replies where Puppet is not idling
   --filter-replies 'ok() && !data("idling")'

   # include only responses where the array item includes 'needle'
   --filter-replies 'ok() && include(data("array"), "needle")'


`

	r.cmd = cli.app.Command("req", help)
	r.cmd.Alias("rpc")

	r.cmd.Arg("agent", "The agent to invoke").Required().StringVar(&r.agent)
	r.cmd.Arg("action", "The action to invoke").Required().StringVar(&r.action)
	r.cmd.Arg("args", "Arguments to pass to the action in key=val format").StringMapVar(&r.args)
	r.cmd.Flag("json", "Produce JSON output only").Short('j').BoolVar(&r.jsonOnly)
	r.cmd.Flag("table", "Produce a Table output of successful responses").BoolVar(&r.tableOnly)

	r.fo = discovery.NewStandardOptions()
	r.fo.AddFilterFlags(r.cmd)
	r.fo.AddFlatFileFlags(r.cmd)
	r.fo.AddSelectionFlags(r.cmd)

	r.cmd.Flag("limit", "Limits request to a set of targets eg 10 or 10%").StringVar(&r.limit)
	r.cmd.Flag("limit-seed", "Seed value for deterministic random limits").PlaceHolder("SEED").Int64Var(&r.limitSeed)
	r.cmd.Flag("batch", "Do requests in batches").PlaceHolder("SIZE").IntVar(&r.batch)
	r.cmd.Flag("batch-sleep", "Sleep time between batches").PlaceHolder("SECONDS").IntVar(&r.batchSleep)
	r.cmd.Flag("workers", "How many workers to start for receiving messages").Default("3").IntVar(&r.workers)
	r.cmd.Flag("np", "Disable the progress bar").BoolVar(&r.noProgress)
	r.cmd.Flag("verbose", "Enable verbose output").Short('v').BoolVar(&r.verbose)
	r.cmd.Flag("display", "Display only a subset of results (ok, failed, all, none)").EnumVar(&r.displayOverride, "ok", "failed", "all", "none")
	r.cmd.Flag("output-file", "Filename to write output to").PlaceHolder("FILENAME").Short('o').StringVar(&r.outputFile)
	r.cmd.Flag("filter-replies", "Filter replies using a expr filter").PlaceHolder("EXPR").StringVar(&r.exprFilter)

	return
}

func (r *reqCommand) parseFilterOptions() (*protocol.Filter, error) {
	return r.fo.NewFilter(r.agent)
}

func (r *reqCommand) configureProgressBar(count int, expected int) {
	if r.noProgress {
		return
	}

	width := c.ProgressWidth()
	if width == -1 {
		r.noProgress = true
		fmt.Printf("\nInvoking %s#%s action\n\n", r.agent, r.action)

		return
	}

	r.progressBar = uiprogress.AddBar(count).AppendCompleted().PrependElapsed()
	r.progressBar.Width = width

	fmt.Println()

	r.progressBar.PrependFunc(func(b *uiprogress.Bar) string {
		if b.Current() < expected {
			return c.Colorize("red", "%d / %d", b.Current(), count)
		}

		return c.Colorize("green", "%d / %d", b.Current(), count)
	})

	uiprogress.Start()
}

func (r *reqCommand) responseHandler(results *replyfmt.RPCResults) func(pr protocol.Reply, reply *rpc.RPCReply) {
	return func(pr protocol.Reply, reply *rpc.RPCReply) {
		mu.Lock()
		defer mu.Unlock()

		if r.progressBar != nil {
			r.progressBar.Incr()
		}

		if reply != nil {
			results.Replies = append(results.Replies, &replyfmt.RPCReply{Sender: pr.SenderID(), RPCReply: reply})
		}
	}
}

func (r *reqCommand) prepareConfiguration() (err error) {
	r.ddl, err = agentddl.Find(r.agent, cfg.LibDir)
	if err != nil {
		return fmt.Errorf("could not find DDL for agent %s: %s", r.agent, err)
	}

	r.actionInterface, err = r.ddl.ActionInterface(r.action)
	if err != nil {
		return err
	}

	r.input, _, err = r.actionInterface.ValidateAndConvertToDDLTypes(r.args)
	if err != nil {
		return fmt.Errorf("invalid input: %s", err)
	}

	r.filter, err = r.parseFilterOptions()
	if err != nil {
		return fmt.Errorf("could not parse filters: %s", err)
	}

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
		r.noProgress = true
	}

	r.fo.SetDefaultsFromChoria(c)

	return nil
}

func (r *reqCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	r.startTime = time.Now()

	err = r.prepareConfiguration()
	if err != nil {
		return err
	}
	defer r.outputWriter.Flush()

	dstart := time.Now()
	nodes, err := r.discover()
	if err != nil {
		return fmt.Errorf("could not discover nodes: %s", err)
	}
	dend := time.Now()

	expected := len(nodes)
	if expected == 0 {
		return fmt.Errorf("did not discover any nodes")
	}

	results := &replyfmt.RPCResults{
		Agent:   r.agent,
		Action:  r.action,
		Replies: []*replyfmt.RPCReply{},
	}

	opts := []rpc.RequestOption{
		rpc.Collective(r.fo.Collective),
		rpc.Targets(nodes),
		rpc.ReplyHandler(r.responseHandler(results)),
		rpc.Workers(r.workers),
		rpc.LimitMethod(cfg.RPCLimitMethod),
		rpc.ReplyExprFilter(r.exprFilter),
		rpc.DiscoveryEndCB(func(d, l int) error {
			r.configureProgressBar(l, expected)

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

	rpcres, err := agent.Do(ctx, r.action, r.input, opts...)
	if err != nil {
		return fmt.Errorf("could not perform request: %s", err)
	}

	results.Stats = rpcres.Stats()
	results.Stats.OverrideDiscoveryTime(dstart, dend)

	if !r.noProgress {
		uiprogress.Stop()
		fmt.Println()
	}

	err = r.displayResults(results)
	if err != nil {
		return fmt.Errorf("could not display results: %s", err)
	}

	return
}

func (r *reqCommand) displayResults(res *replyfmt.RPCResults) error {
	defer r.outputWriter.Flush()

	if r.jsonOnly {
		return res.RenderJSON(r.outputWriter, r.actionInterface)
	}

	if r.tableOnly {
		return res.RenderTable(r.outputWriter, r.actionInterface)
	}

	mode := replyfmt.DisplayDDL
	switch r.displayOverride {
	case "ok":
		mode = replyfmt.DisplayOK
	case "failed":
		mode = replyfmt.DisplayFailed
	case "all":
		mode = replyfmt.DisplayAll
	case "none":
		mode = replyfmt.DisplayNone
	}

	return res.RenderTXT(r.outputWriter, r.actionInterface, r.verbose, r.silent, mode, c.Config.Color, c.Logger("req"))
}

func (r *reqCommand) Configure() error {
	protocol.ClientStrictValidation = false

	return commonConfigure()
}

func (r *reqCommand) discover() ([]string, error) {
	nodes, _, err := r.fo.Discover(ctx, c, r.agent, true, !r.silent, c.Logger("discovery"))
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func init() {
	cli.commands = append(cli.commands, &reqCommand{})
}
