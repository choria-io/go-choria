// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package discover

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/choria-io/appbuilder/builder"
	"github.com/choria-io/fisk"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/appbuilder"
	"github.com/sirupsen/logrus"
)

type Command struct {
	StandardFilter bool                       `json:"std_filters"`
	Filter         *discovery.StandardOptions `json:"filter"`
	builder.GenericCommand
	builder.GenericSubCommands
}

type Discover struct {
	b         *builder.AppBuilder
	cmd       *fisk.CmdClause
	fo        *discovery.StandardOptions
	def       *Command
	cfg       any
	arguments map[string]any
	flags     map[string]any
	json      bool
	log       builder.Logger
	ctx       context.Context
}

func NewDiscoverCommand(b *builder.AppBuilder, j json.RawMessage, log builder.Logger) (builder.Command, error) {
	find := &Discover{
		arguments: map[string]any{},
		flags:     map[string]any{},
		def:       &Command{},
		cfg:       b.Configuration(),
		ctx:       b.Context(),
		log:       log,
		b:         b,
	}

	err := json.Unmarshal(j, find.def)
	if err != nil {
		return nil, err
	}

	return find, nil
}

func Register() error {
	return builder.RegisterCommand("discover", NewDiscoverCommand)
}

func MustRegister() {
	builder.MustRegisterCommand("discover", NewDiscoverCommand)
}

func (r *Discover) Validate(log builder.Logger) error { return nil }

func (r *Discover) String() string { return fmt.Sprintf("%s (discover)", r.def.Name) }

func (r *Discover) SubCommands() []json.RawMessage {
	return r.def.Commands
}

func (r *Discover) CreateCommand(app builder.KingpinCommand) (*fisk.CmdClause, error) {
	r.cmd = builder.CreateGenericCommand(app, &r.def.GenericCommand, r.arguments, r.flags, r.b, r.runCommand)

	r.fo = discovery.NewStandardOptions()

	if r.def.StandardFilter {
		r.fo.AddFilterFlags(r.cmd)
		r.fo.AddFlatFileFlags(r.cmd)
		r.fo.AddSelectionFlags(r.cmd)
	}

	r.cmd.Flag("json", "Produce JSON output").BoolVar(&r.json)

	return r.cmd, nil
}

func (r *Discover) runCommand(_ *fisk.ParseContext) error {
	cfg, err := config.NewConfig(choria.UserConfig())
	if err != nil {
		return err
	}

	logger, ok := any(r.log).(*logrus.Logger)
	if ok {
		cfg.CustomLogger = logger
	}

	fw, err := choria.NewWithConfig(cfg)
	if err != nil {
		return err
	}

	log := fw.Logger("find")

	if r.def.Filter != nil {
		err = appbuilder.ProcessStdDiscoveryOptions(r.def.Filter, r.arguments, r.flags, r.cfg)
		if err != nil {
			return err
		}

		r.fo.Merge(r.def.Filter)
	}

	r.fo.SetDefaultsFromChoria(fw)

	targets, _, err := r.fo.Discover(r.ctx, fw, "discovery", true, false, log)
	if err != nil {
		return err
	}

	if r.json {
		out, err := json.MarshalIndent(targets, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	for _, t := range targets {
		fmt.Println(t)
	}

	return nil
}
