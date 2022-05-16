// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package appbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/nats.go"
	"gopkg.in/alecthomas/kingpin.v2"
)

type KVCommand struct {
	Action     string `json:"action"`
	Bucket     string `json:"bucket"`
	Key        string `json:"key"`
	Value      string `json:"value"`
	RenderJSON bool   `json:"json"`

	StandardSubCommands
	StandardCommand
}

type KV struct {
	b         *AppBuilder
	Arguments map[string]*string
	Flags     map[string]*string
	cmd       *kingpin.CmdClause
	def       *KVCommand
	cfg       interface{}
	ctx       context.Context
}

func NewKVCommand(b *AppBuilder, j json.RawMessage) (*KV, error) {
	kv := &KV{
		def:       &KVCommand{},
		cfg:       b.cfg,
		ctx:       b.ctx,
		b:         b,
		Arguments: map[string]*string{},
		Flags:     map[string]*string{},
	}

	err := json.Unmarshal(j, kv.def)
	if err != nil {
		return nil, err
	}

	return kv, nil
}

func (r *KV) SubCommands() []json.RawMessage {
	return r.def.Commands
}

func (r *KV) CreateCommand(app inter.FlagApp) (*kingpin.CmdClause, error) {
	r.cmd = app.Command(r.def.Name, r.def.Description).Action(r.b.runWrapper(r.def.StandardCommand, r.runCommand))
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

	if r.def.Action == "get" || r.def.Action == "history" && !r.def.RenderJSON {
		r.cmd.Flag("json", "Renders results in JSON format").BoolVar(&r.def.RenderJSON)
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

func (r *KV) getAction(kv nats.KeyValue) error {
	entry, err := kv.Get(r.def.Key)
	if err != nil {
		return err
	}

	if r.def.RenderJSON {
		ej, err := json.MarshalIndent(r.entryMap(entry), "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(ej))
	} else {
		fmt.Println(string(entry.Value()))
	}

	return nil
}

func (r *KV) putAction(kv nats.KeyValue) error {
	v, err := parseStateTemplate(r.def.Value, r.Arguments, r.Flags, r.cfg)
	if err != nil {
		return err
	}

	rev, err := kv.PutString(r.def.Key, v)
	if err != nil {
		return err
	}

	fmt.Printf("Wrote revision %d\n", rev)

	return nil
}

func (r *KV) delAction(kv nats.KeyValue) error {
	err := kv.Delete(r.def.Key)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted key %s\n", r.def.Key)

	return nil
}

func (r *KV) opStringForOp(kvop nats.KeyValueOp) string {
	var op string

	switch kvop {
	case nats.KeyValuePurge:
		op = "PURGE"
	case nats.KeyValueDelete:
		op = "DELETE"
	case nats.KeyValuePut:
		op = "PUT"
	default:
		op = kvop.String()
	}

	return op
}

func (r *KV) entryMap(e nats.KeyValueEntry) map[string]interface{} {
	if e == nil {
		return nil
	}

	res := map[string]interface{}{
		"operation": r.opStringForOp(e.Operation()),
		"revision":  e.Revision(),
		"value":     util.Base64IfNotPrintable(e.Value()),
		"created":   e.Created().Unix(),
	}

	return res
}

func (r *KV) historyAction(kv nats.KeyValue) error {
	history, err := kv.History(r.def.Key)
	if err != nil {
		return err
	}

	if r.def.RenderJSON {
		hist := map[string]map[string]map[string]interface{}{}
		for _, e := range history {
			if _, ok := hist[e.Bucket()]; !ok {
				hist[e.Bucket()] = map[string]map[string]interface{}{}
			}

			hist[e.Bucket()][e.Key()] = r.entryMap(e)
		}

		j, err := json.MarshalIndent(hist, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(j))
		return nil
	}

	table := util.NewUTF8Table("Seq", "Operation", "Time", "Length", "Value")
	for _, e := range history {
		val := util.Base64IfNotPrintable(e.Value())
		if len(val) > 40 {
			val = fmt.Sprintf("%s...%s", val[0:15], val[len(val)-15:])
		}

		table.AddRow(e.Revision(), r.opStringForOp(e.Operation()), e.Created().Format(time.RFC822), len(e.Value()), val)
	}

	fmt.Println(table.Render())

	return nil
}

func (r *KV) runCommand(_ *kingpin.ParseContext) error {
	fw, err := choria.New(choria.UserConfig())
	if err != nil {
		return err
	}

	kv, err := fw.KV(r.ctx, nil, r.def.Bucket, false)
	if err != nil {
		return err
	}

	switch r.def.Action {
	case "get":
		err = r.getAction(kv)

	case "put":
		err = r.putAction(kv)

	case "del":
		err = r.delAction(kv)

	case "history":
		err = r.historyAction(kv)
	}

	return err
}
