// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package appbuilder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/choria"
	"gopkg.in/alecthomas/kingpin.v2"
)

type KVCommand struct {
	Action string `json:"action"`
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
	Value  string `json:"value"`

	StandardSubCommands
	StandardCommand
}

type KV struct {
	Arguments map[string]*string
	Flags     map[string]*string
	cmd       *kingpin.CmdClause
	def       *KVCommand
	cfg       interface{}
	ctx       context.Context
}

func NewKVCommand(ctx context.Context, j json.RawMessage, cfg interface{}) (*KV, error) {
	kv := &KV{
		def: &KVCommand{},
		cfg: cfg,
		ctx: ctx,
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

func (r *KV) CreateCommand(app kingpinParent) (*kingpin.CmdClause, error) {
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
		entry, err := kv.Get(r.def.Key)
		if err != nil {
			return err
		}
		fmt.Println(string(entry.Value()))

	case "put":
		v, err := parseStateTemplate(r.def.Value, r.Arguments, r.Flags, r.cfg)
		if err != nil {
			return err
		}

		rev, err := kv.PutString(r.def.Key, v)
		if err != nil {
			return err
		}

		fmt.Printf("Wrote revision %d\n", rev)

	case "del":
		err = kv.Delete(r.def.Key)
		if err != nil {
			return err
		}
		fmt.Printf("Deleted key %s\n", r.def.Key)
	}

	return nil
}
