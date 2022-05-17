// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package appbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/choria-io/go-choria/config"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/inter"
	"gopkg.in/alecthomas/kingpin.v2"
)

type DiscoverCommand struct {
	StandardFilter bool                       `json:"std_filters"`
	Filter         *discovery.StandardOptions `json:"filter"`
	StandardCommand
	StandardSubCommands
}

type Discover struct {
	b         *AppBuilder
	cmd       *kingpin.CmdClause
	fo        *discovery.StandardOptions
	def       *DiscoverCommand
	cfg       interface{}
	arguments map[string]*string
	flags     map[string]*string
	json      bool
	log       *logrus.Entry
	ctx       context.Context
}

func NewDiscoverCommand(b *AppBuilder, j json.RawMessage, log *logrus.Entry) (*Discover, error) {
	find := &Discover{
		arguments: map[string]*string{},
		flags:     map[string]*string{},
		def:       &DiscoverCommand{},
		cfg:       b.cfg,
		ctx:       b.ctx,
		log:       log,
		b:         b,
	}

	err := json.Unmarshal(j, find.def)
	if err != nil {
		return nil, err
	}

	return find, nil
}

func (r *Discover) SubCommands() []json.RawMessage {
	return r.def.Commands
}

func (r *Discover) CreateCommand(app inter.FlagApp) (*kingpin.CmdClause, error) {
	r.cmd = createStandardCommand(app, r.b, &r.def.StandardCommand, r.arguments, r.flags, r.runCommand)

	r.fo = discovery.NewStandardOptions()

	if r.def.StandardFilter {
		r.fo.AddFilterFlags(r.cmd)
		r.fo.AddFlatFileFlags(r.cmd)
		r.fo.AddSelectionFlags(r.cmd)
	}

	r.cmd.Flag("json", "Produce JSON output").BoolVar(&r.json)

	return r.cmd, nil
}

func (r *Discover) runCommand(_ *kingpin.ParseContext) error {
	cfg, err := config.NewConfig(choria.UserConfig())
	if err != nil {
		return err
	}
	cfg.CustomLogger = r.log.Logger

	fw, err := choria.NewWithConfig(cfg)
	if err != nil {
		return err
	}

	log := fw.Logger("find")

	if r.def.Filter != nil {
		err = processStdDiscoveryOptions(r.def.Filter, r.arguments, r.flags, r.cfg)
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
