package client

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"

	addl "github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
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

func (a *agent) ActionRequiredInputs(act string) map[string]*addl.ActionInputItem {
	inputs := make(map[string]*addl.ActionInputItem)

	for _, act := range a.DDL.Actions {
		for name, input := range act.Input {
			if !input.Optional {
				inputs[name] = input
			}
		}
	}

	return inputs
}

func (g *Generator) funcMap() template.FuncMap {
	choriaTypeToGo := func(v string) string {
		switch v {
		case "string", "list":
			return "string"
		case "integer":
			return "int64"
		case "number", "float":
			return "float64"
		case "boolean":
			return "bool"
		case "hash":
			return "map[string]interface{}"
		case "array":
			return "[]interface{}"
		default:
			return "interface{}"
		}
	}

	choriaOptionalInputsToFuncArgs := func(act *addl.Action) string {
		inputs := g.optionalInputSelect(act, false)
		parts := []string{}

		for name, input := range inputs {
			goType := choriaTypeToGo(input.Type)
			parts = append(parts, fmt.Sprintf("%sInput %s", strings.ToLower(name), goType))
		}

		return strings.Join(parts, ", ")
	}

	return template.FuncMap{
		"GeneratedWarning": func() string {
			return fmt.Sprintf("// generated code; DO NOT EDIT")
		},
		"Base64Encode": func(v string) string { return base64.StdEncoding.EncodeToString([]byte(v)) },
		"Capitalize":   strings.Title,
		"ToLower":      strings.ToLower,
		"SnakeToCamel": func(v string) string {
			parts := strings.Split(v, "_")
			out := []string{}
			for _, s := range parts {
				out = append(out, strings.Title(s))
			}

			return strings.Join(out, "")
		},
		"ChoriaRequiredInputs": func(act *addl.Action) map[string]*addl.ActionInputItem {
			return g.optionalInputSelect(act, true)
		},
		"ChoriaOptionalInputs": func(act *addl.Action) map[string]*addl.ActionInputItem {
			return g.optionalInputSelect(act, false)
		},
		"ChoriaOptionalInputsToFuncArgs": choriaOptionalInputsToFuncArgs,
		"ChoriaTypeToGoType":             choriaTypeToGo,
		"ChoriaTypeToValOfType": func(v string) string {
			switch v {
			case "string", "list":
				return "val.(string)"
			case "integer":
				return "val.(int64)"
			case "number", "float":
				return "val.(float64)"
			case "boolean":
				return "val.(bool)"
			case "hash":
				return "val.(map[string]interface{})"
			case "array":
				return "val.([]interface{})"
			default:
				return "val.(interface{})"
			}
		},
	}
}

func (g *Generator) writeActions() error {
	code, err := base64.StdEncoding.DecodeString(templates["action"])
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
		RequiredInputs    map[string]*addl.ActionInputItem
		OptionalInputs    map[string]*addl.ActionInputItem
		Outputs           map[string]*addl.ActionOutputItem
	}

	for _, actname := range g.agent.DDL.ActionNames() {
		actint, err := g.agent.DDL.ActionInterface(actname)
		if err != nil {
			return err
		}

		outfile := path.Join(g.OutDir, fmt.Sprintf("action_%s.go", actint.Name))
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

		err = g.formatGoSource(outfile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) writeBasics() error {
	for _, t := range []string{"resultdetails", "requestor", "ddl", "discover", "rpcoptions", "client", "initoptions", "logging", "doc"} {
		outfile := path.Join(g.OutDir, t+".go")
		logrus.Infof("Writing %s", outfile)
		out, err := os.Create(outfile)
		if err != nil {
			return err
		}

		code, err := base64.StdEncoding.DecodeString(templates[t])
		if err != nil {
			return err
		}

		tpl := template.Must(template.New(t).Funcs(g.funcMap()).Parse(string(code)))

		err = tpl.Execute(out, g.agent)
		if err != nil {
			return err
		}

		err = g.formatGoSource(outfile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) formatGoSource(f string) error {
	bs, err := ioutil.ReadFile(f)
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
	return ioutil.WriteFile(f, bs, os.ModePerm)
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

	raw, err := ioutil.ReadFile(g.DDLFile)
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

func (g *Generator) optionalInputSelect(action *addl.Action, opt bool) map[string]*addl.ActionInputItem {
	inputs := make(map[string]*addl.ActionInputItem)

	for name, act := range action.Input {
		if act.Optional == opt {
			inputs[name] = act
		}
	}

	return inputs
}
