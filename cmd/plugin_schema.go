// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	addl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

type pSchemaCommand struct {
	command
	kind   string
	plugin string
	format string
}

func (g *pSchemaCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("plugin"); ok {
		g.cmd = tool.Cmd().Command("schema", "Retrieves the Schema for a specified plugin")
		g.cmd.Arg("kind", "The kind of schema to retrieve").Required().EnumVar(&g.kind, "agent")
		g.cmd.Arg("plugin", "The name of the plugin to retrieve").Required().StringVar(&g.plugin)
		g.cmd.Flag("format", "Renders the schema in a different format").Default("json").EnumVar(&g.format, "json", "ddl")
	}

	return nil
}

func (g *pSchemaCommand) Configure() error {
	return commonConfigure()
}

func (g *pSchemaCommand) renderJSON(ddl []byte) error {
	buf := bytes.NewBuffer([]byte{})
	json.Compact(buf, ddl)

	fmt.Println(buf.String())

	return nil
}

func (g *pSchemaCommand) renderDDL(ddl []byte) error {
	if g.kind != "agent" {
		return fmt.Errorf("can not render DDL %s/%s in DDL format", g.kind, g.plugin)
	}

	agent, err := addl.NewFromBytes(ddl)
	if err != nil {
		return err
	}

	ddls, err := agent.ToRuby()
	if err != nil {
		return err
	}

	fmt.Println(ddls)

	return nil
}

func (g *pSchemaCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	resolvers, err := c.DDLResolvers()
	if err != nil {
		return err
	}

	for _, r := range resolvers {
		ddl, err := r.DDLBytes(ctx, g.kind, g.plugin, c)
		if err != nil {
			continue
		}

		switch g.format {
		case "json":
			return g.renderJSON(ddl)

		case "ddl":
			return g.renderDDL(ddl)
		}

	}

	return fmt.Errorf("did not find schema for %s/%s", g.kind, g.kind)
}

func init() {
	cli.commands = append(cli.commands, &pSchemaCommand{})
}
