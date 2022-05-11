// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package appbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/replyfmt"
	"gopkg.in/alecthomas/kingpin.v2"
)

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
	StandardFilter     bool              `json:"std_filters"`
	OutputFormatsFlags bool              `json:"output_formats_flags"`
	DisplayFlag        bool              `json:"display_flag"`
	BatchFlags         bool              `json:"batch_flags"`
	Arguments          []GenericArgument `json:"arguments"`
	Flags              []RPCFlag         `json:"flags"`
	Request            RPCRequest        `json:"request"`

	StandardCommand
	StandardSubCommands
}

type RPC struct {
	cmd        *kingpin.CmdClause
	fo         *discovery.StandardOptions
	def        *RPCCommand
	cfg        interface{}
	Arguments  map[string]*string
	Flags      map[string]*string
	senders    bool
	json       bool
	table      bool
	display    string
	batch      int
	batchSleep int
	ctx        context.Context
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

func (r *RPC) CreateCommand(app kingpinParent) (*kingpin.CmdClause, error) {
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

	if r.def.OutputFormatsFlags {
		r.cmd.Flag("senders", "List only the names of matching nodes").BoolVar(&r.senders)
		r.cmd.Flag("json", "Render results as JSON").BoolVar(&r.json)
		r.cmd.Flag("table", "Render results as a table").BoolVar(&r.table)
	}

	if r.def.StandardFilter {
		r.fo = discovery.NewStandardOptions()
		r.fo.AddFilterFlags(r.cmd)
		r.fo.AddFlatFileFlags(r.cmd)
		r.fo.AddSelectionFlags(r.cmd)
	}

	if r.def.BatchFlags {
		r.cmd.Flag("batch", "Do requests in batches").PlaceHolder("SIZE").IntVar(&r.batch)
		r.cmd.Flag("batch-sleep", "Sleep time between batches").PlaceHolder("SECONDS").IntVar(&r.batchSleep)
	}

	if r.def.DisplayFlag {
		r.cmd.Flag("display", "Display only a subset of results (ok, failed, all, none)").EnumVar(&r.display, "ok", "failed", "all", "none")
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

func (r *RPC) runCommand(_ *kingpin.ParseContext) error {
	noisy := !(r.json || r.senders)

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

	cmd, inputs, opts, err := r.choriaCommand()
	if err != nil {
		return err
	}
	log.Infof(strings.Join(cmd, " "))

	if r.fo != nil {
		filter, err := r.fo.NewFilter(r.def.Request.Agent)
		if err != nil {
			return err
		}
		r.fo.SetDefaultsFromConfig(fw.Configuration())

		opts = append(opts, client.Filter(filter))
		opts = append(opts, client.Collective(r.fo.Collective))
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
	mu := sync.Mutex{}

	opts = append(opts, client.ReplyHandler(func(pr protocol.Reply, reply *client.RPCReply) {
		mu.Lock()
		if reply != nil {
			results.Replies = append(results.Replies, &replyfmt.RPCReply{Sender: pr.SenderID(), RPCReply: reply})
		}
		mu.Unlock()
	}))

	if noisy {
		opts = append(opts, client.DiscoveryStartCB(func() {
			fmt.Printf("Discovering nodes...")
		}))
		opts = append(opts, client.DiscoveryEndCB(func(discovered int, limited int) error {
			fmt.Printf("%d\n", limited)
			fmt.Println()
			return nil
		}))
	}

	rpcres, err := agent.Do(r.ctx, r.def.Request.Action, rpcInputs, opts...)
	if err != nil {
		return err
	}
	results.Stats = rpcres.Stats()

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
			params = append(params, "-C", opt.Collective)
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
