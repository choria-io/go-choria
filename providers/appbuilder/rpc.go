// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package appbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/replyfmt"
	"github.com/gosuri/uiprogress"
	"gopkg.in/alecthomas/kingpin.v2"
)

// TODO: this is essentially a full re-implementation of choria req, not sure it adds value over
// just doing an exec of choria req.  However I think in time we might extend this to cover some
// new display options to only show some fields etc, but as it stands I am inclined to remove it
// should we get no additional needs in this one.

type RPCFlag struct {
	GenericFlag
	ReplyFilter string `json:"reply_filter"`
}

type RPCRequest struct {
	Agent  string            `json:"agent"`
	Action string            `json:"action"`
	Params map[string]string `json:"params"`
}

type RPCCommand struct {
	StandardFilter    bool                       `json:"std_filters"`
	OutputFormatFlags bool                       `json:"output_format_flags"`
	OutputFormat      string                     `json:"output_format"`
	Display           string                     `json:"display"`
	DisplayFlag       bool                       `json:"display_flag"`
	BatchFlags        bool                       `json:"batch_flags"`
	BatchSize         int                        `json:"batch"`
	BatchSleep        int                        `json:"batch_sleep"`
	NoProgress        bool                       `json:"no_progress"`
	Arguments         []GenericArgument          `json:"arguments"`
	Flags             []RPCFlag                  `json:"flags"`
	Request           RPCRequest                 `json:"request"`
	Filter            *discovery.StandardOptions `json:"filter"`

	StandardCommand
	StandardSubCommands
}

type RPC struct {
	cmd         *kingpin.CmdClause
	fo          *discovery.StandardOptions
	def         *RPCCommand
	cfg         interface{}
	Arguments   map[string]*string
	Flags       map[string]*string
	senders     bool
	json        bool
	table       bool
	display     string
	batch       int
	batchSleep  int
	progressBar *uiprogress.Bar
	ctx         context.Context
}

func NewRPCCommand(ctx context.Context, j json.RawMessage, cfg interface{}) (*RPC, error) {
	rpc := &RPC{
		Arguments: map[string]*string{},
		Flags:     map[string]*string{},
		def:       &RPCCommand{},
		cfg:       cfg,
		ctx:       ctx,
	}

	err := json.Unmarshal(j, rpc.def)
	if err != nil {
		return nil, err
	}

	return rpc, nil
}

func (r *RPC) SubCommands() []json.RawMessage {
	return r.def.Commands
}

func (r *RPC) CreateCommand(app inter.FlagApp) (*kingpin.CmdClause, error) {
	r.cmd = app.Command(r.def.Name, r.def.Description).Action(r.runCommand)
	for _, a := range r.def.Aliases {
		r.cmd.Alias(a)
	}

	for _, a := range r.def.Arguments {
		arg := r.cmd.Arg(a.Name, a.Description)
		if a.Required {
			arg.Required()
		}

		r.Arguments[a.Name] = arg.String()
	}

	switch {
	case r.def.OutputFormatFlags && r.def.OutputFormat != "":
		return nil, fmt.Errorf("only one of output_format_flags and output_format may be supplied to command %s", r.def.Name)

	case r.def.OutputFormatFlags:
		r.cmd.Flag("senders", "List only the names of matching nodes").BoolVar(&r.senders)
		r.cmd.Flag("json", "Render results as JSON").BoolVar(&r.json)
		r.cmd.Flag("table", "Render results as a table").BoolVar(&r.table)

	case r.def.OutputFormat == "senders":
		r.senders = true

	case r.def.OutputFormat == "json":
		r.json = true

	case r.def.OutputFormat == "table":
		r.table = true

	case r.def.OutputFormat != "":
		return nil, fmt.Errorf("invalid output format %q, valid formats are senders, json and table", r.def.OutputFormat)
	}

	switch {
	case r.def.StandardFilter && r.def.Filter != nil:
		return nil, fmt.Errorf("only one of std_filters and filter may be supplied in command %s", r.def.Name)

	case r.def.StandardFilter:
		r.fo = discovery.NewStandardOptions()
		r.fo.AddFilterFlags(r.cmd)
		r.fo.AddFlatFileFlags(r.cmd)
		r.fo.AddSelectionFlags(r.cmd)

	case r.def.Filter != nil:
		r.fo = r.def.Filter
	}

	switch {
	case r.def.BatchFlags && r.def.BatchSize > 0:
		return nil, fmt.Errorf("only one of batch_flags and batch may be supplied in command %s", r.def.Name)

	case r.def.BatchFlags:
		r.cmd.Flag("batch", "Do requests in batches").PlaceHolder("SIZE").IntVar(&r.batch)
		r.cmd.Flag("batch-sleep", "Sleep time between batches").PlaceHolder("SECONDS").IntVar(&r.batchSleep)

	case r.def.BatchSize > 0:
		r.batch = r.def.BatchSize
		r.batchSleep = r.def.BatchSleep
	}

	switch {
	case r.def.DisplayFlag && r.def.Display != "":
		return nil, fmt.Errorf("only one of display_flag and display may be supplied in command %s", r.def.Name)

	case r.def.DisplayFlag:
		r.cmd.Flag("display", "Display only a subset of results (ok, failed, all, none)").EnumVar(&r.display, "ok", "failed", "all", "none")

	case r.def.Display != "":
		r.display = r.def.Display
	}

	for _, f := range r.def.Flags {
		flag := r.cmd.Flag(f.Name, f.Description)
		if f.Required {
			flag.Required()
		}
		if f.PlaceHolder != "" {
			flag.PlaceHolder(f.PlaceHolder)
		}
		r.Flags[f.Name] = flag.String()
	}

	return r.cmd, nil
}

func (r *RPC) configureProgressBar(fw inter.Framework, count int, expected int) {
	if r.def.NoProgress {
		return
	}

	width := fw.ProgressWidth()
	if width == -1 {
		fmt.Printf("\nInvoking %s#%s action\n\n", r.def.Request.Agent, r.def.Request.Action)
		return
	}

	r.progressBar = uiprogress.AddBar(count).AppendCompleted().PrependElapsed()
	r.progressBar.Width = width

	fmt.Println()

	r.progressBar.PrependFunc(func(b *uiprogress.Bar) string {
		if b.Current() < expected {
			return fw.Colorize("red", "%d / %d", b.Current(), count)
		}

		return fw.Colorize("green", "%d / %d", b.Current(), count)
	})

	uiprogress.Start()
}

func (r *RPC) runCommand(_ *kingpin.ParseContext) error {
	var (
		noisy   = !(r.json || r.senders || r.def.NoProgress)
		mu      = sync.Mutex{}
		dt      time.Duration
		targets []string
	)

	fw, err := choria.New(choria.UserConfig())
	if err != nil {
		return err
	}

	log := fw.Logger(r.def.Name)

	agent, err := client.New(fw, r.def.Request.Agent)
	if err != nil {
		return err
	}

	err = agent.ResolveDDL(r.ctx)
	if err != nil {
		return err
	}

	ddl := agent.DDL()
	action, err := ddl.ActionInterface(r.def.Request.Action)
	if err != nil {
		return err
	}

	// todo: surface rpc command somewhere as learning aid
	_, inputs, opts, err := r.choriaCommand()
	if err != nil {
		return err
	}

	rpcInputs, _, err := action.ValidateAndConvertToDDLTypes(inputs)
	if err != nil {
		return err
	}

	results := &replyfmt.RPCResults{
		Agent:   r.def.Request.Agent,
		Action:  r.def.Request.Action,
		Replies: []*replyfmt.RPCReply{},
	}

	opts = append(opts, client.ReplyHandler(func(pr protocol.Reply, reply *client.RPCReply) {
		mu.Lock()
		if reply != nil {
			results.Replies = append(results.Replies, &replyfmt.RPCReply{Sender: pr.SenderID(), RPCReply: reply})
			if r.progressBar != nil {
				r.progressBar.Incr()
			}
		}
		mu.Unlock()
	}))

	start := time.Now()

	if r.fo == nil {
		r.fo = discovery.NewStandardOptions()
	}

	r.fo.SetDefaultsFromChoria(fw)
	targets, dt, err = r.fo.Discover(r.ctx, fw, r.def.Request.Agent, true, noisy, log)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no nodes discovered")
	}
	opts = append(opts, client.Targets(targets))

	if noisy {
		if ddl.Metadata.Service {
			r.configureProgressBar(fw, 1, 1)
		} else {
			r.configureProgressBar(fw, len(targets), len(targets))
		}
	}

	rpcres, err := agent.Do(r.ctx, r.def.Request.Action, rpcInputs, opts...)
	if err != nil {
		return err
	}
	results.Stats = rpcres.Stats()

	if dt > 0 {
		rpcres.Stats().OverrideDiscoveryTime(start, start.Add(dt))
	}

	if r.progressBar != nil {
		uiprogress.Stop()
		fmt.Println()
	}

	switch {
	case r.senders:
		err = results.RenderNames(os.Stdout, r.json, false)
	case r.table:
		err = results.RenderTable(os.Stdout, action)
	case r.json:
		err = results.RenderJSON(os.Stdout, action)
	default:
		mode := replyfmt.DisplayDDL
		switch r.display {
		case "ok":
			mode = replyfmt.DisplayOK
		case "failed":
			mode = replyfmt.DisplayFailed
		case "all":
			mode = replyfmt.DisplayAll
		case "none":
			mode = replyfmt.DisplayNone
		}

		err = results.RenderTXT(os.Stdout, action, false, false, mode, fw.Configuration().Color, log)
	}
	if err != nil {
		return err
	}

	return nil

}

func (r *RPC) choriaCommand() (cmd []string, inputs map[string]string, opts []client.RequestOption, err error) {
	var params []string
	opts = []client.RequestOption{}
	inputs = map[string]string{}

	for k, v := range r.def.Request.Params {
		body, err := r.parseStateTemplate(v)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(body) > 0 {
			params = append(params, fmt.Sprintf("%s=%s", k, body))
			inputs[k] = body
		}
	}

	if r.senders {
		params = append(params, "--senders")
	}
	if r.json {
		params = append(params, "--json")
	}
	if r.table {
		params = append(params, "--table")
	}
	if r.display != "" {
		params = append(params, "--display", r.display)
	}

	if r.batch > 0 {
		opts = append(opts, client.InBatches(r.batch, r.batchSleep))
		params = append(params, "--batch", fmt.Sprintf("%d", r.batch))
		if r.batchSleep > 0 {
			params = append(params, "--batch-sleep", fmt.Sprintf("%d", r.batchSleep))

		}
	}

	if r.fo != nil {
		opt := r.fo
		if opt.DynamicDiscoveryTimeout {
			params = append(params, "--discovery-window")
		}
		if opt.Collective != "" {
			params = append(params, "-T", opt.Collective)
		}
		for _, f := range opt.AgentFilter {
			params = append(params, "-A", f)
		}
		for _, f := range opt.ClassFilter {
			params = append(params, "-C", f)
		}
		for _, f := range opt.FactFilter {
			params = append(params, "-F", f)
		}
		for _, f := range opt.CombinedFilter {
			params = append(params, "-W", f)
		}
		for _, f := range opt.IdentityFilter {
			params = append(params, "-I", f)
		}
		for k, v := range opt.DiscoveryOptions {
			params = append(params, "--do", fmt.Sprintf("%s=%s", k, v))
		}
		if opt.NodesFile != "" {
			params = append(params, "--nodes", opt.NodesFile)
		}
		if opt.CompoundFilter != "" {
			params = append(params, "-S", opt.CompoundFilter)
		}

		if opt.DiscoveryMethod != "" {
			params = append(params, "--dm", opt.DiscoveryMethod)
		}
	}

	filter := ""
	for _, flag := range r.def.Flags {
		if *r.Flags[flag.Name] != "" {
			if flag.ReplyFilter == "" {
				continue
			}

			if filter != "" {
				return nil, nil, nil, fmt.Errorf("only one filter flag can match")
			}

			body, err := r.parseStateTemplate(flag.ReplyFilter)
			if err != nil {
				return nil, nil, nil, err
			}

			filter = body
			break
		}
	}

	cmd = []string{"choria", "req", r.def.Request.Agent, r.def.Request.Action}

	cmd = append(cmd, params...)
	if filter != "" {
		opts = append(opts, client.ReplyExprFilter(filter))
		cmd = append(cmd, "--filter-replies", filter)
	}

	return cmd, inputs, opts, nil
}

func (r *RPC) parseStateTemplate(body string) (string, error) {
	return parseStateTemplate(body, r.Arguments, r.Flags, r.cfg)
}
