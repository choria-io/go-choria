// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package kv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/choria-io/appbuilder/builder"
	"github.com/choria-io/go-choria/config"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/nats.go"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Command struct {
	Action     string                    `json:"action"`
	Bucket     string                    `json:"bucket"`
	Key        string                    `json:"key"`
	Value      string                    `json:"value"`
	RenderJSON bool                      `json:"json"`
	Transform  *builder.GenericTransform `json:"transform"`

	builder.GenericCommand
	builder.GenericSubCommands
}

type KV struct {
	b         *builder.AppBuilder
	Arguments map[string]*string
	Flags     map[string]*string
	cmd       *kingpin.CmdClause
	def       *Command
	cfg       interface{}
	log       builder.Logger
	ctx       context.Context
}

func NewKVCommand(b *builder.AppBuilder, j json.RawMessage, log builder.Logger) (builder.Command, error) {
	kv := &KV{
		def:       &Command{},
		cfg:       b.Configuration(),
		ctx:       b.Context(),
		b:         b,
		log:       log,
		Arguments: map[string]*string{},
		Flags:     map[string]*string{},
	}

	err := json.Unmarshal(j, kv.def)
	if err != nil {
		return nil, err
	}

	return kv, nil
}

func Register() error {
	return builder.RegisterCommand("kv", NewKVCommand)
}

func MustRegister() {
	builder.MustRegisterCommand("kv", NewKVCommand)
}

func (r *KV) Validate(log builder.Logger) error {
	if r.def.Type != "kv" {
		return fmt.Errorf("not a kv command")
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

	if r.def.Bucket == "" {
		errs = append(errs, "bucket is required")
	}

	if r.def.Key == "" {
		errs = append(errs, "key is required")
	}

	act := r.def.Action
	if act == "put" && r.def.Value == "" {
		errs = append(errs, "value is required for put operations")
	}

	if !(act == "put" || act == "get" || act == "history" || act == "del") {
		errs = append(errs, fmt.Sprintf("invalid action %q", act))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ", "))
	}

	return nil
}

func (r *KV) String() string { return fmt.Sprintf("%s (kv)", r.def.Name) }

func (r *KV) SubCommands() []json.RawMessage {
	return r.def.Commands
}

func (r *KV) CreateCommand(app builder.KingpinCommand) (*kingpin.CmdClause, error) {
	r.cmd = builder.CreateGenericCommand(app, &r.def.GenericCommand, r.Arguments, r.Flags, r.b.Configuration(), r.runCommand)

	if r.def.Action == "get" || r.def.Action == "history" && !r.def.RenderJSON {
		r.cmd.Flag("json", "Renders results in JSON format").BoolVar(&r.def.RenderJSON)
	}

	return r.cmd, nil
}

func (r *KV) getAction(kv nats.KeyValue) error {
	entry, err := kv.Get(r.def.Key)
	if err != nil {
		return err
	}

	switch {
	case r.def.Transform != nil:
		r.def.Transform.FTransformJSON(r.ctx, os.Stdout, entry.Value())
	case r.def.RenderJSON:
		ej, err := json.MarshalIndent(r.entryMap(entry), "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(ej))
	default:
		fmt.Println(string(entry.Value()))
	}

	return nil
}

func (r *KV) putAction(kv nats.KeyValue) error {
	v, err := builder.ParseStateTemplate(r.def.Value, r.Arguments, r.Flags, r.cfg)
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
		hist := map[string]map[string][]interface{}{}
		for _, e := range history {
			if _, ok := hist[e.Bucket()]; !ok {
				hist[e.Bucket()] = map[string][]interface{}{}
			}

			hist[e.Bucket()][e.Key()] = append(hist[e.Bucket()][e.Key()], r.entryMap(e))
		}

		j, err := json.MarshalIndent(hist, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(j))
		return nil
	}

	table := util.NewUTF8Table("Seq", "Operation", "Time", "Value")
	for _, e := range history {
		val := util.Base64IfNotPrintable(e.Value())
		if len(val) > 40 {
			val = fmt.Sprintf("%s...%s", val[0:15], val[len(val)-15:])
		}

		table.AddRow(e.Revision(), r.opStringForOp(e.Operation()), e.Created().Format(time.RFC822), val)
	}

	fmt.Println(table.Render())

	return nil
}

func (r *KV) runCommand(_ *kingpin.ParseContext) error {
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
