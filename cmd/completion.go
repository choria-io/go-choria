// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/fs"
	agents "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

type completionCommand struct {
	command

	showZsh    bool
	showBash   bool
	list       string
	agent      string
	action     string
	zshScript  string
	bashScript string
}

func (e *completionCommand) Setup() error {
	e.cmd = cli.app.Command("completion", "Shell completion support").Hidden()
	e.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)
	e.cmd.Flag("zsh", "ZSH completion script").BoolVar(&e.showZsh)
	e.cmd.Flag("bash", "Bash completion script").Default("true").BoolVar(&e.showBash)
	e.cmd.Flag("list", "List various discovered items").EnumVar(&e.list, "agents", "actions", "inputs")
	e.cmd.Flag("agent", "Limit to a specific agent").StringVar(&e.agent)
	e.cmd.Flag("action", "Limit to a specific action").StringVar(&e.action)

	return nil
}

func (e *completionCommand) Configure() error {
	err = commonConfigure()
	if err != nil {
		cfg, err = config.NewDefaultConfig()
		if err != nil {
			return err
		}
		cfg.Choria.SecurityProvider = "file"
	}

	// we dont want to invoke names against the network constantly
	// so we disable the registry network features and just take
	// whats in the cache
	cfg.RegistryCacheOnly = true
	cfg.DisableSecurityProviderVerify = true

	s, err := fs.FS.ReadFile("completion/zsh.template")
	if err != nil {
		return err
	}
	e.zshScript = string(s)

	s, err = fs.FS.ReadFile("completion/bash.template")
	if err != nil {
		return err
	}
	e.bashScript = string(s)
	return err
}

func (e *completionCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	switch e.list {
	case "agents":
		e.listAgents()

	case "actions":
		if e.agent == "" {
			return fmt.Errorf("please indicate an agent to list actions for")
		}

		e.listActions()

	case "inputs":
		if e.agent == "" {
			return fmt.Errorf("please indicate an agent to list inputs for")
		}

		if e.action == "" {
			return fmt.Errorf("please indicate an action to list inputs for")
		}

		e.listInputs()

	default:
		switch {
		case e.showZsh:
			fmt.Println(e.zshScript)

		case e.showBash:
			fmt.Println(e.bashScript)
		}
	}

	return nil
}

func (e *completionCommand) listInputs() {
	ddl, err := e.loadAgent(e.agent)
	if err != nil {
		return
	}

	act, err := ddl.ActionInterface(e.action)
	if err != nil {
		return
	}

	found := []string{}

	for _, i := range act.InputNames() {
		input, _ := act.GetInput(i)

		switch {
		case e.showZsh:
			found = append(found, fmt.Sprintf("%s:%s", i, input.Description))
		case e.showBash:
			found = append(found, i)
		}
	}

	sort.Strings(found)
	fmt.Println(strings.Join(found, "\n"))
}

func (e *completionCommand) listActions() {
	found := []string{}

	ddl, err := e.loadAgent(e.agent)
	if err != nil {
		return
	}

	for _, act := range ddl.Actions {
		switch {
		case e.showZsh:
			found = append(found, fmt.Sprintf("%s:%s", act.Name, act.Description))
		case e.showBash:
			found = append(found, act.Name)
		}
	}

	sort.Strings(found)
	fmt.Println(strings.Join(found, "\n"))
}

func (e *completionCommand) loadAgent(name string) (*agents.DDL, error) {
	resolvers, err := c.DDLResolvers()
	if err != nil {
		return nil, err
	}

	log := c.Logger("ddl")
	for _, resolver := range resolvers {
		log.Infof("Resolving DDL agent/%s via %s", name, resolver)
		data, err := resolver.DDLBytes(ctx, "agent", name, c)
		if err == nil {
			return agents.NewFromBytes(data)
		}
	}

	return nil, fmt.Errorf("agent/%s ddl not found", name)
}

func (e *completionCommand) listAgents() {
	found := []string{}
	known := map[string]struct{}{}

	resolvers, err := c.DDLResolvers()
	if err != nil {
		return
	}
	for _, resolver := range resolvers {
		names, err := resolver.DDLNames(ctx, "agent", c)
		if err != nil {
			return
		}

		for _, name := range names {
			known[name] = struct{}{}
		}
	}

	for name := range known {
		ddl, err := e.loadAgent(name)
		if err != nil {
			return
		}

		switch {
		case e.showZsh:
			found = append(found, fmt.Sprintf("%s:%s", ddl.Metadata.Name, ddl.Metadata.Description))
		case e.showBash:
			found = append(found, ddl.Metadata.Name)
		}
	}

	sort.Strings(found)
	fmt.Println(strings.Join(found, "\n"))
}

func init() {
	cli.commands = append(cli.commands, &completionCommand{})
}
