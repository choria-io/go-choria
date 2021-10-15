// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/choria-io/go-choria/internal/fs"
	addl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"

	"github.com/sirupsen/logrus"
	"golang.org/x/tools/imports"
)

type Generator struct {
	agent *agent

	DDLFile     string
	OutDir      string
	PackageName string
}

type agent struct {
	Package string
	DDL     *addl.DDL
	RawDDL  string // raw text of the JSON DDL file
}

func (a *agent) ActionRequiredInputs(act string) map[string]*common.InputItem {
	inputs := make(map[string]*common.InputItem)

	for _, act := range a.DDL.Actions {
		for name, input := range act.Input {
			if !input.Optional {
				inputs[name] = input
			}
		}
	}

	return inputs
}

func (g *Generator) writeActions() error {
	code, err := fs.FS.ReadFile("client/action.templ")
	if err != nil {
		return err
	}

	type action struct {
		Agent             *agent
		Package           string
		AgentName         string
		ActionName        string
		ActionDescription string
		OutputNames       []string
		InputNames        []string
		RequiredInputs    map[string]*common.InputItem
		OptionalInputs    map[string]*common.InputItem
		Outputs           map[string]*common.OutputItem
	}

	for _, actname := range g.agent.DDL.ActionNames() {
		actint, err := g.agent.DDL.ActionInterface(actname)
		if err != nil {
			return err
		}

		outfile := filepath.Join(g.OutDir, fmt.Sprintf("action_%s.go", actint.Name))
		logrus.Infof("Writing %s for action %s", outfile, actint.Name)

		out, err := os.Create(outfile)
		if err != nil {
			return err
		}
		defer out.Close()

		act := &action{
			Agent:             g.agent,
			Package:           g.agent.Package,
			AgentName:         g.agent.DDL.Metadata.Name,
			ActionName:        actint.Name,
			ActionDescription: actint.Description,
			InputNames:        actint.InputNames(),
			OutputNames:       actint.OutputNames(),
			RequiredInputs:    g.optionalInputSelect(actint, false),
			OptionalInputs:    g.optionalInputSelect(actint, true),
			Outputs:           actint.Output,
		}

		tpl := template.Must(template.New(actint.Name).Funcs(g.funcMap()).Parse(string(code)))
		err = tpl.Execute(out, act)
		if err != nil {
			return err
		}

		err = FormatGoSource(outfile)
		if err != nil {
			return err
		}
	}

	ddlPath := filepath.Join(g.OutDir, "ddl.json")
	cDDL := bytes.NewBuffer([]byte{})
	json.Compact(cDDL, []byte(g.agent.RawDDL))
	logrus.Infof("Writing %s", ddlPath)
	err = os.WriteFile(filepath.Join(g.OutDir, "ddl.json"), cDDL.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) writeBasics() error {
	dir, err := fs.FS.ReadDir("client")
	if err != nil {
		return err
	}

	for _, file := range dir {
		t := strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name()))
		if t == "action" {
			continue
		}

		outfile := path.Join(g.OutDir, t+".go")
		logrus.Infof("Writing %s", outfile)
		out, err := os.Create(outfile)
		if err != nil {
			return err
		}

		code, err := fs.FS.ReadFile(filepath.Join("client", file.Name()))
		if err != nil {
			return err
		}

		tpl := template.Must(template.New(t).Funcs(g.funcMap()).Parse(string(code)))

		err = tpl.Execute(out, g.agent)
		if err != nil {
			return err
		}

		err = FormatGoSource(outfile)
		if err != nil {
			return err
		}
	}

	return nil
}

func FormatGoSource(f string) error {
	bs, err := os.ReadFile(f)
	if err != nil {
		return err
	}
	opt := imports.Options{
		Comments:   true,
		FormatOnly: true,
	}
	bs, err = imports.Process(f, bs, &opt)
	if err != nil {
		return err
	}
	return os.WriteFile(f, bs, os.ModePerm)
}

func (g *Generator) GenerateClient() error {
	var err error

	g.agent = &agent{}
	g.agent.DDL, err = addl.New(g.DDLFile)
	if err != nil {
		return err
	}

	if g.agent.DDL == nil {
		return fmt.Errorf("could not find any DDL")
	}

	raw, err := os.ReadFile(g.DDLFile)
	if err != nil {
		return err
	}

	g.agent.RawDDL = string(raw)
	g.agent.Package = g.PackageName

	if g.PackageName == "" {
		g.agent.Package = strings.ToLower(g.agent.DDL.Metadata.Name) + "client"
	}

	logrus.Infof("Writing Choria Client for Agent %s Version %s to %s", g.agent.DDL.Metadata.Name, g.agent.DDL.Metadata.Version, g.OutDir)
	err = g.writeActions()
	if err != nil {
		return err
	}

	err = g.writeBasics()
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) optionalInputSelect(action *addl.Action, opt bool) map[string]*common.InputItem {
	inputs := make(map[string]*common.InputItem)

	for name, act := range action.Input {
		if act.Optional == opt {
			inputs[name] = act
		}
	}

	return inputs
}
