// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/appbuilder/builder"
	"github.com/choria-io/fisk"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/replyfmt"
	"github.com/choria-io/go-choria/providers/appbuilder"
	"github.com/gosuri/uiprogress"
	"github.com/sirupsen/logrus"
)

type Flag struct {
	builder.GenericFlag
	ReplyFilter string `json:"reply_filter"`
}

type Request struct {
	Agent  string                     `json:"agent"`
	Action string                     `json:"action"`
	Params map[string]string          `json:"inputs"`
	Filter *discovery.StandardOptions `json:"filter"`
}

type Command struct {
	StandardFilter        bool               `json:"std_filters"`
	OutputFormatFlags     bool               `json:"output_format_flags"`
	OutputFormat          string             `json:"output_format"`
	Display               string             `json:"display"`
	DisplayFlag           bool               `json:"display_flag"`
	BatchFlags            bool               `json:"batch_flags"`
	BatchSize             int                `json:"batch"`
	BatchSleep            int                `json:"batch_sleep"`
	NoProgress            bool               `json:"no_progress"`
	AllNodesConfirmPrompt string             `json:"all_nodes_confirm_prompt"`
	Flags                 []Flag             `json:"flags"`
	Request               Request            `json:"request"`
	Transform             *builder.Transform `json:"transform"`

	builder.GenericCommand
	builder.GenericSubCommands
}

type RPC struct {
	b           *builder.AppBuilder
	cmd         *fisk.CmdClause
	fo          *discovery.StandardOptions
	def         *Command
	cfg         interface{}
	arguments   map[string]interface{}
	flags       map[string]interface{}
	senders     bool
	json        bool
	table       bool
	display     string
	batch       int
	batchSleep  int
	progressBar *uiprogress.Bar
	log         builder.Logger
	ctx         context.Context
}

func NewRPCCommand(b *builder.AppBuilder, j json.RawMessage, log builder.Logger) (builder.Command, error) {
	rpc := &RPC{
		arguments: map[string]interface{}{},
		flags:     map[string]interface{}{},
		def:       &Command{},
		cfg:       b.Configuration(),
		ctx:       b.Context(),
		b:         b,
		log:       log,
	}

	err := json.Unmarshal(j, rpc.def)
	if err != nil {
		return nil, err
	}

	return rpc, nil
}

func Register() error {
	return builder.RegisterCommand("rpc", NewRPCCommand)
}

func MustRegister() {
	builder.MustRegisterCommand("rpc", NewRPCCommand)
}

func (r *RPC) String() string { return fmt.Sprintf("%s (rpc)", r.def.Name) }

func (r *RPC) Validate(log builder.Logger) error {
	if r.def.Type != "rpc" {
		return fmt.Errorf("not a rpc command")
	}

	var errs []string

	err := r.def.GenericCommand.Validate(log)
	if err != nil {
		errs = append(errs, err.Error())
	}

	if r.def.Transform != nil {
		err := r.def.Transform.Validate(log)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	if r.def.Request.Agent == "" {
		errs = append(errs, "agent is required")
	}
	if r.def.Request.Action == "" {
		errs = append(errs, "action is required")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ", "))
	}

	return nil
}

func (r *RPC) SubCommands() []json.RawMessage {
	return r.def.Commands
}

func (r *RPC) CreateCommand(app builder.KingpinCommand) (*fisk.CmdClause, error) {
	r.cmd = builder.CreateGenericCommand(app, &r.def.GenericCommand, r.arguments, r.flags, r.b, r.runCommand)

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

	if r.def.StandardFilter {
		r.fo = discovery.NewStandardOptions()
		r.fo.AddFilterFlags(r.cmd)
		r.fo.AddFlatFileFlags(r.cmd)
		r.fo.AddSelectionFlags(r.cmd)
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
		r.flags[f.Name] = flag.String()
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

func (r *RPC) setupFilter(fw inter.Framework) error {
	var err error

	if r.fo == nil {
		r.fo = discovery.NewStandardOptions()
	}

	if r.def.Request.Filter != nil {
		err = appbuilder.ProcessStdDiscoveryOptions(r.def.Request.Filter, r.arguments, r.flags, r.cfg)
		if err != nil {
			return err
		}

		r.fo.Merge(r.def.Request.Filter)
	}

	r.fo.SetDefaultsFromChoria(fw)

	if r.def.AllNodesConfirmPrompt != "" && r.fo.NodesFile == "" {
		f, err := r.fo.NewFilter(r.def.Request.Agent)
		if err != nil {
			return err
		}
		if f.Empty() {
			ans := false
			err := survey.AskOne(&survey.Confirm{Message: r.def.AllNodesConfirmPrompt, Default: false}, &ans)
			if err != nil {
				return err
			}
			if !ans {
				return fmt.Errorf("aborted")
			}
		}
	}

	return nil
}

func (r *RPC) runCommand(_ *fisk.ParseContext) error {
	var (
		noisy   = !(r.json || r.senders || r.def.NoProgress || r.def.Transform != nil)
		mu      = sync.Mutex{}
		dt      time.Duration
		targets []string
	)

	cfg, err := config.NewConfig(choria.UserConfig())
	if err != nil {
		return err
	}

	logger, ok := interface{}(r.log).(*logrus.Logger)
	if ok {
		cfg.CustomLogger = logger
	}

	fw, err := choria.NewWithConfig(cfg)
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

	_, rpcInputs, opts, err := r.reqOptions(action)
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

	err = r.setupFilter(fw)
	if err != nil {
		return err
	}

	start := time.Now()
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

	if r.batch > 0 {
		if r.batchSleep == 0 {
			r.batchSleep = 1
		}

		opts = append(opts, client.InBatches(r.batch, r.batchSleep))
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

	err = r.renderResults(fw, log, results, action)
	if err != nil {
		return err
	}

	return nil

}

func (r *RPC) transformResults(w io.Writer, results *replyfmt.RPCResults, action *agent.Action) error {
	out := bytes.NewBuffer([]byte{})
	err := results.RenderJSON(out, action)
	if err != nil {
		return err
	}

	res, err := r.def.Transform.TransformBytes(r.ctx, out.Bytes(), r.flags, r.arguments, r.b.Configuration())
	if err != nil {
		return err
	}

	fmt.Fprintln(w, string(res))
	return nil
}

func (r *RPC) renderResults(fw inter.Framework, log *logrus.Entry, results *replyfmt.RPCResults, action *agent.Action) (err error) {
	switch {
	case r.def.Transform != nil:
		err = r.transformResults(os.Stdout, results, action)
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

	return err
}

func (r *RPC) reqOptions(action *agent.Action) (inputs map[string]string, rpcInputs map[string]interface{}, opts []client.RequestOption, err error) {
	opts = []client.RequestOption{}
	inputs = map[string]string{}

	for k, v := range r.def.Request.Params {
		body, err := r.parseStateTemplate(v)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(body) > 0 {
			inputs[k] = body
		}
	}

	filter := ""
	for _, flag := range r.def.Flags {
		if r.flags[flag.Name] != "" {
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

	if filter != "" {
		opts = append(opts, client.ReplyExprFilter(filter))
	}

	rpcInputs, _, err = action.ValidateAndConvertToDDLTypes(inputs)
	if err != nil {
		return nil, nil, nil, err
	}

	return inputs, rpcInputs, opts, nil
}

func (r *RPC) parseStateTemplate(body string) (string, error) {
	return builder.ParseStateTemplate(body, r.arguments, r.flags, r.cfg)
}
