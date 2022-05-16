// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package appbuilder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/AlecAivazis/survey/v2"
	"github.com/adrg/xdg"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/fs"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/alessio/shellescape.v1"
)

type StandardCommand struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Aliases       []string          `json:"aliases"`
	Type          string            `json:"type"`
	Arguments     []GenericArgument `json:"args"`
	Flags         []GenericFlag     `json:"flags"`
	ConfirmPrompt string            `json:"confirm_prompt"`
}

type StandardSubCommands struct {
	Commands []json.RawMessage `json:"commands"`
}

type Definition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Author      string `json:"author"`

	StandardSubCommands

	commands []command
}

type GenericArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type GenericFlag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	PlaceHolder string `json:"placeholder"`
}

type templateState struct {
	Arguments interface{}
	Flags     interface{}
	Config    interface{}
}

type command interface {
	CreateCommand(app inter.FlagApp) (*kingpin.CmdClause, error)
	SubCommands() []json.RawMessage
}

type AppBuilder struct {
	ctx  context.Context
	def  *Definition
	name string
	cfg  map[string]interface{}
	log  *logrus.Entry
}

var (
	errDefinitionNotfound = errors.New("definition not found")
	appDefPattern         = "%s-app.yaml"
)

func NewAppBuilder(ctx context.Context, name string) *AppBuilder {
	builder := &AppBuilder{
		ctx:  ctx,
		name: name,
	}

	return builder
}

func (b *AppBuilder) RunCommand() {
	err := b.runCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Choria application %s: %v\n", b.name, err)
		os.Exit(1)
	}
}

func (b *AppBuilder) runCLI() error {
	logger := logrus.New()
	b.log = logrus.NewEntry(logger)
	logger.SetLevel(logrus.WarnLevel)
	if os.Getenv("BUILDER_DEBUG") != "" {
		logger.SetLevel(logrus.DebugLevel)
	}

	var err error

	b.def, err = b.loadDefinition(b.name)
	if err != nil {
		return err
	}

	err = b.loadConfig()
	if err != nil {
		return err
	}

	cmd := kingpin.New(b.name, b.def.Description)
	cmd.Version(b.def.Version)
	cmd.Author(b.def.Author)
	cmd.VersionFlag.Hidden()

	err = b.registerCommands(cmd, b.def.commands...)
	if err != nil {
		return err
	}

	_, err = cmd.Parse(os.Args[1:])
	return err
}

func (b *AppBuilder) registerCommands(cli inter.FlagApp, cmds ...command) error {
	for _, c := range cmds {
		cmd, err := c.CreateCommand(cli)
		if err != nil {
			return err
		}

		subs := c.SubCommands()
		if len(subs) > 0 {
			for _, sub := range subs {
				subCommand, err := b.createCommand(sub)
				if err != nil {
					return err
				}

				err = b.registerCommands(cmd, subCommand)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (b *AppBuilder) HasDefinition() bool {
	source, _ := b.findConfigFile(fmt.Sprintf(appDefPattern, b.name), "BUILDER_APP")
	if source == "" {
		return false
	}

	return util.FileExist(source)
}

func (b *AppBuilder) loadDefinition(name string) (*Definition, error) {
	source, err := b.findConfigFile(fmt.Sprintf(appDefPattern, name), "BUILDER_APP")
	if err != nil {
		return nil, errDefinitionNotfound
	}

	if b.log != nil {
		b.log.Infof("Loading application definition %v", source)
	}

	cfg, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	d := &Definition{}
	cfgj, err := yaml.YAMLToJSON(cfg)
	if err != nil {
		return nil, err
	}

	if os.Getenv("BUILDER_NOVALIDATE") == "" {
		schema, err := fs.FS.ReadFile("schemas/builder.json")
		if err != nil {
			return nil, fmt.Errorf("could not load schema: %v", err)
		}

		sloader := gojsonschema.NewBytesLoader(schema)
		dloader := gojsonschema.NewBytesLoader(cfgj)
		result, err := gojsonschema.Validate(sloader, dloader)
		if err != nil {
			return nil, fmt.Errorf("schema validation failed: %s", err)
		}

		if !result.Valid() {
			fmt.Printf("The Builder Application %s does not pass validation against https://choria.io/schemas/choria/builder/v1/application.json:\n\n", source)
			for _, err := range result.Errors() {
				fmt.Printf(" - %s\n", err)
			}

			return nil, fmt.Errorf("validation failed")
		}
	}

	err = json.Unmarshal(cfgj, d)
	if err != nil {
		return nil, err
	}

	return d, b.createCommands(d, d.Commands)
}

func (b *AppBuilder) createCommands(d *Definition, defs []json.RawMessage) error {
	for _, c := range defs {
		cmd, err := b.createCommand(c)
		if err != nil {
			return err
		}

		d.commands = append(d.commands, cmd)
	}

	return nil
}

func (b *AppBuilder) createCommand(def json.RawMessage) (command, error) {
	t := gjson.GetBytes(def, "type")
	if !t.Exists() {
		return nil, fmt.Errorf("command does not have a type\n%s", string(def))
	}

	switch t.String() {
	case "rpc":
		return NewRPCCommand(b, def)
	case "parent":
		return NewParentCommand(b, def)
	case "kv":
		return NewKVCommand(b, def)
	case "exec":
		return NewExecCommand(b, def)
	case "discover":
		return NewDiscoverCommand(b, def)
	default:
		return nil, fmt.Errorf("unknown command type %q", t.String())
	}
}

func (b *AppBuilder) runWrapper(cmd StandardCommand, handler kingpin.Action) kingpin.Action {
	return func(pc *kingpin.ParseContext) error {
		if cmd.ConfirmPrompt != "" {
			ans := false
			err := survey.AskOne(&survey.Confirm{Message: cmd.ConfirmPrompt, Default: false}, &ans)
			if err != nil {
				return err
			}
			if !ans {
				return fmt.Errorf("aborted")
			}
		}

		return handler(pc)
	}
}

func (b *AppBuilder) findConfigFile(name string, env string) (string, error) {
	sources := []string{
		filepath.Join(xdg.ConfigHome, "choria", "builder"),
		"/etc/choria/builder",
	}

	cur, err := filepath.Abs(".")
	if err == nil {
		sources = append([]string{cur}, sources...)
	}

	if b.log != nil {
		b.log.Debugf("Searching for app definition %s in %v", name, sources)
	}

	source := os.Getenv(env)

	if source == "" {
		for _, s := range sources {
			path := filepath.Join(s, name)
			if choria.FileExist(path) {
				source = path
				break
			}
		}
	}

	if source == "" {
		return "", fmt.Errorf("could not find configuration %s in %s", name, strings.Join(sources, ", "))
	}

	return source, nil
}

func (b *AppBuilder) loadConfig() error {
	source, err := b.findConfigFile("applications.yaml", "BUILDER_CONFIG")
	if err != nil {
		return nil
	}

	b.log.Debugf("Loading configuration file %s", source)

	cfgb, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	cfgj, err := yaml.YAMLToJSON(cfgb)
	if err != nil {
		return err
	}

	b.cfg = map[string]interface{}{}

	return json.Unmarshal(cfgj, &b.cfg)
}

func parseStateTemplate(body string, args interface{}, flags interface{}, cfg interface{}) (string, error) {
	state := templateState{
		Arguments: args,
		Flags:     flags,
		Config:    cfg,
	}

	funcs := map[string]interface{}{
		"require": func(v interface{}, reason string) (interface{}, error) {
			err := errors.New("value required")
			if reason != "" {
				err = errors.New(reason)
			}

			switch val := v.(type) {
			case string:
				if val == "" {
					return "", err
				}
			default:
				if v == nil {
					return "", err
				}
			}

			return v, nil
		},
		"escape": func(v string) string {
			return shellescape.Quote(v)
		},
		"read_file": func(v string) (string, error) {
			b, err := os.ReadFile(v)
			if err != nil {
				return "", err
			}

			return string(b), nil
		},
		"default": func(v string, dflt string) string {
			if v != "" {
				return v
			}

			return dflt
		},
	}

	temp, err := template.New("choria").Funcs(funcs).Parse(body)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = temp.Execute(&b, state)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func createStandardCommand(app inter.FlagApp, b *AppBuilder, sc *StandardCommand, arguments map[string]*string, flags map[string]*string, cb kingpin.Action) *kingpin.CmdClause {
	cmd := app.Command(sc.Name, sc.Description).Action(b.runWrapper(*sc, cb))
	for _, a := range sc.Aliases {
		cmd.Alias(a)
	}

	if arguments != nil {
		for _, a := range sc.Arguments {
			arg := cmd.Arg(a.Name, a.Description)
			if a.Required {
				arg.Required()
			}

			arguments[a.Name] = arg.String()
		}
	}

	if flags != nil {
		for _, f := range sc.Flags {
			flag := cmd.Flag(f.Name, f.Description)
			if f.Required {
				flag.Required()
			}
			if f.PlaceHolder != "" {
				flag.PlaceHolder(f.PlaceHolder)
			}
			flags[f.Name] = flag.String()
		}
	}

	return cmd
}

func processStdDiscoveryOptions(f *discovery.StandardOptions, arguments interface{}, flags interface{}, config interface{}) error {
	var err error

	if f.Collective != "" {
		f.Collective, err = parseStateTemplate(f.Collective, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	if f.NodesFile != "" {
		f.NodesFile, err = parseStateTemplate(f.NodesFile, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	if f.CompoundFilter != "" {
		f.CompoundFilter, err = parseStateTemplate(f.CompoundFilter, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.CombinedFilter {
		f.CombinedFilter[i], err = parseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.IdentityFilter {
		f.IdentityFilter[i], err = parseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.AgentFilter {
		f.AgentFilter[i], err = parseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.ClassFilter {
		f.ClassFilter[i], err = parseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.FactFilter {
		f.FactFilter[i], err = parseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	return nil
}
