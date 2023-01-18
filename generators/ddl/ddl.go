// Copyright (c) 2019-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ddl

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	iu "github.com/choria-io/go-choria/internal/util"
	ddl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
	"github.com/choria-io/go-choria/server/agents"
)

type Generator struct {
	JSONOut      string
	RubyOut      string
	SkipVerify   bool
	ForceConvert bool
}

func (c *Generator) ValidateJSON(agent *ddl.DDL) error {
	// new validation library wants to only handle pure json type while the previous would take
	// *agent.DDL and figure it out, now we have to take data and convert it to a basic type before
	// validation
	jd, err := json.Marshal(agent)
	if err != nil {
		return err
	}
	var d any
	err = json.Unmarshal(jd, &d)
	if err != nil {
		return err
	}

	errs, err := iu.ValidateSchemaFromFS("schemas/mcorpc/ddl/v1/agent.json", d)
	if err != nil {
		return err
	}
	if len(errs) != 0 {
		fmt.Printf("The generate DDL does not pass validation against https://choria.io/schemas/mcorpc/ddl/v1/agent.json:\n\n")
		for _, err := range errs {
			fmt.Printf(" - %s\n", err)
		}

		return fmt.Errorf("JSON DDL validation failed")
	}

	return nil
}

func (c *Generator) ConvertToRuby() error {
	jddl, err := ddl.New(c.JSONOut)
	if err != nil {
		return err
	}

	if !c.SkipVerify {
		fmt.Println("Validating JSON DDL against the schema...")
		err = c.ValidateJSON(jddl)
		if err != nil {
			fmt.Printf("\nWARN: DDL does not pass JSON Schema Validation: %s\n", err)
		}
		fmt.Println()
	}

	rddl, err := jddl.ToRuby()
	if err != nil {
		return err
	}

	return os.WriteFile(c.RubyOut, []byte(rddl), 0644)
}

func (c *Generator) GenerateDDL() error {
	agent := &ddl.DDL{
		Schema:   "https://choria.io/schemas/mcorpc/ddl/v1/agent.json",
		Metadata: &agents.Metadata{},
		Actions:  []*ddl.Action{},
	}

	fmt.Println(`
Choria Agents need a DDL file that describes the facilities provided by an
agent, these files include:

* Metadata about the agent such as who made it and its license
* Every known action
  * Every input the action expects and its types, help and how to show it
  * Every output the action produce and its types, help and how to show it
  * How to summarize the returned outputs

This tool assists in generating such a DDL file by interactively asking you questions.
The JSON file is saved regularly after every major section of input, at any time
"if you press ^C you'll get a partial JSON DDL with what you have already provided.

These files are in JSON format and have a scheme, if you configure your editor
to consume the schema you'll have a convenient way to modify the file after.
	`)

	survey.AskOne(&survey.Input{Message: "Press enter to start"}, &struct{}{})

	err := c.askMetaData(agent)
	if err != nil {
		return err
	}

	err = c.saveDDL(agent)
	if err != nil {
		return err
	}

	err = c.askActions(agent)
	if err != nil {
		return err
	}

	err = c.saveDDL(agent)
	if err != nil {
		return err
	}

	if !c.SkipVerify {
		fmt.Println("Validating JSON DDL against the schema...")
		err = c.ValidateJSON(agent)
		if err != nil {
			fmt.Printf("WARN: DDL does not pass JSON Schema Validation: %s\n", err)
		}
		fmt.Println()
	}

	return nil
}

func (c *Generator) saveDDL(agent *ddl.DDL) error {
	err := c.saveJSON(agent)
	if err != nil {
		return err
	}

	return c.saveRuby(agent)
}

func (c *Generator) saveRuby(agent *ddl.DDL) error {
	if c.RubyOut == "" {
		return nil
	}

	out, err := os.Create(c.RubyOut)
	if err != nil {
		return err
	}
	defer out.Close()

	r, err := agent.ToRuby()
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(out, r)
	return err
}

func (c *Generator) saveJSON(agent *ddl.DDL) error {
	out, err := os.Create(c.JSONOut)
	if err != nil {
		return err
	}
	defer out.Close()

	j, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(out, string(j))
	return err
}

func (c *Generator) askBasicItem(name string, prompt string, help string, t survey.Transformer, v survey.Validator) *survey.Question {
	return &survey.Question{
		Name:      name,
		Prompt:    &survey.Input{Message: prompt, Help: help},
		Validate:  v,
		Transform: t,
	}
}

func (c *Generator) AskBool(m string) bool {
	should := false
	prompt := &survey.Confirm{
		Message: m,
	}
	survey.AskOne(prompt, &should)
	return should

}

func (c *Generator) askEnum(name string, prompt string, help string, valid []string, v survey.Validator) *survey.Question {
	return &survey.Question{
		Name:     name,
		Prompt:   &survey.Select{Message: prompt, Help: help, Options: valid},
		Validate: v,
	}
}

func (c *Generator) showJSON(m string, d any) error {
	j, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(m)
	fmt.Println()
	fmt.Println(string(j))
	fmt.Println()

	return nil
}

func (c *Generator) urlValidator(v any) error {
	err := survey.Required(v)
	if err != nil {
		return err
	}

	vs, ok := v.(string)
	if !ok {
		return fmt.Errorf("should be a string")
	}

	u, err := url.ParseRequestURI(vs)
	if !(err == nil && u.Scheme != "" && u.Host != "") {
		return fmt.Errorf("is not a valid url")
	}

	return nil
}

func (c *Generator) semVerValidator(v any) error {
	err := survey.Required(v)
	if err != nil {
		return err
	}

	vs, ok := v.(string)
	if !ok {
		return fmt.Errorf("should be a string")
	}

	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(vs) {
		return fmt.Errorf("must be basic semver x.y.z format")
	}

	return nil
}

func (c *Generator) boolValidator(v any) error {
	vs, ok := v.(string)
	if !ok {
		return fmt.Errorf("should be a string")
	}

	if vs == "" {
		return nil
	}

	_, err := iu.StrToBool(vs)
	return err
}

func (c *Generator) shortnameValidator(v any) error {
	err := survey.Required(v)
	if err != nil {
		return err
	}

	vs, ok := v.(string)
	if !ok {
		return fmt.Errorf("should be a string")
	}

	if !regexp.MustCompile(`^[a-z0-9_]*$`).MatchString(vs) {
		return fmt.Errorf("must match ^[a-z0-9_]*$")
	}

	return nil
}

func (c *Generator) askActions(agent *ddl.DDL) error {
	addAction := func() error {
		action := &ddl.Action{
			Input:       make(map[string]*common.InputItem),
			Output:      make(map[string]*common.OutputItem),
			Aggregation: []ddl.ActionAggregateItem{},
		}

		qs := []*survey.Question{
			c.askBasicItem("name", "Action Name", "", survey.ToLower, func(v any) error {
				act := v.(string)

				if act == "" {
					return fmt.Errorf("an action name is required")
				}

				if agent.HaveAction(act) {
					return fmt.Errorf("already have an action %s", act)
				}

				return c.shortnameValidator(v)
			}),

			c.askBasicItem("description", "Description", "", nil, survey.Required),
			c.askEnum("display", "Display Hint", "", []string{"ok", "failed", "always"}, survey.Required),
		}

		err := survey.Ask(qs, action)
		if err != nil {
			return err
		}

		agent.Actions = append(agent.Actions, action)

		err = c.saveDDL(agent)
		if err != nil {
			return err
		}

		fmt.Println(`
Arguments that you pass to an action are called inputs, an action can have
any number of inputs - some being optional and some being required.

         Name: The name of the input argument
       Prompt: A short prompt to show when asking people this information
  Description: A 1 line description about this input
    Data Type: The type of data that this input must hold
     Optional: If this input is required or not
	  Default: A default value when the input is not provided

For string data there are additional properties:

   Max Length: How long a string may be, 0 for unlimited
   Validation: How to validate the string data
		`)

		for {
			fmt.Println()

			if len(action.InputNames()) > 0 {
				fmt.Printf("Existing Inputs: %s\n\n", strings.Join(action.InputNames(), ", "))
			}

			if !c.AskBool("Add an input?") {
				break
			}

			input := &common.InputItem{}
			name := ""
			survey.AskOne(&survey.Input{Message: "Input Name:"}, &name, survey.WithValidator(survey.Required), survey.WithValidator(func(v any) error {
				i := v.(string)
				if i == "" {
					return fmt.Errorf("input name is required")
				}

				_, ok := action.Input[i]
				if ok {
					return fmt.Errorf("input %s already exist", i)
				}

				return c.shortnameValidator(v)
			}))
			qs := []*survey.Question{
				c.askBasicItem("prompt", "Prompt", "", nil, survey.Required),
				c.askBasicItem("description", "Description", "", nil, survey.Required),
				c.askEnum("type", "Data Type", "", []string{"integer", "number", "float", "string", "boolean", "list", "hash", "array"}, survey.Required),
				c.askBasicItem("optional", "Optional (t/f)", "", nil, survey.Required),
			}

			err = survey.Ask(qs, input)
			if err != nil {
				return err
			}

			if input.Type == "string" {
				qs = []*survey.Question{
					c.askBasicItem("maxlength", "Max Length", "", nil, survey.Required),
					c.askEnum("validation", "Validation", "", []string{"shellsafe", "ipv4address", "ipv6address", "ipaddress", "regex"}, survey.Required),
				}
				err = survey.Ask(qs, input)
				if err != nil {
					return err
				}

				if input.Validation == "regex" {
					survey.AskOne(&survey.Input{Message: "Validation Regular Expression"}, &input.Validation, survey.WithValidator(survey.Required))
				}

			} else if input.Type == "list" {
				valid := ""
				prompt := &survey.Input{
					Message: "Valid Values (comma separated)",
					Help:    "List of valid values for this input separated by commas",
				}
				err = survey.AskOne(prompt, &valid, survey.WithValidator(survey.Required))
				if err != nil {
					return err
				}

				input.Enum = strings.Split(valid, ",")
			}

			deflt := ""
			err = survey.AskOne(&survey.Input{Message: "Default Value"}, &deflt)
			if err != nil {
				return err
			}
			if deflt != "" {
				input.Default, err = common.ValToDDLType(input.Type, deflt)
				if err != nil {
					return fmt.Errorf("default for %s does not validate: %s", name, err)
				}
			}

			action.Input[name] = input

			err = c.saveDDL(agent)
			if err != nil {
				return err
			}
		}

		fmt.Println(`
Results that an action produce are called outputs, an action can have
any number of outputs.

         Name: The name of the output
  Description: A 1 line description about this output
    Data Type: The type of data that this output must hold
   Display As: Hint to user interface on a heading to use for this data
	Default: A default value when the output is not provided

		`)

		for {
			fmt.Println()

			if len(action.OutputNames()) > 0 {
				fmt.Printf("Existing Outputs: %s\n\n", strings.Join(action.OutputNames(), ", "))
			}

			if !c.AskBool("Add an output?") {
				break
			}

			output := &common.OutputItem{}
			name := ""
			survey.AskOne(&survey.Input{Message: "Name:"}, &name, survey.WithValidator(survey.Required), survey.WithValidator(func(v any) error {
				i := v.(string)
				if i == "" {
					return fmt.Errorf("output name is required")
				}

				_, ok := action.Output[i]
				if ok {
					return fmt.Errorf("output %s already exist", i)
				}

				return c.shortnameValidator(v)
			}))
			qs := []*survey.Question{
				c.askBasicItem("description", "Description", "", nil, survey.Required),
				c.askEnum("type", "Data Type", "", []string{"integer", "number", "float", "string", "boolean", "list", "hash", "array"}, survey.Required),
				c.askBasicItem("displayas", "Display As", "", nil, survey.Required),
			}

			err = survey.Ask(qs, output)
			if err != nil {
				return err
			}

			deflt := ""
			err = survey.AskOne(&survey.Input{Message: "Default Value"}, &deflt)
			if err != nil {
				return err
			}

			if deflt != "" {
				output.Default, err = common.ValToDDLType(output.Type, deflt)
				if err != nil {
					return fmt.Errorf("default for %s does not validate: %s", name, err)
				}
			}

			action.Output[name] = output
			err = c.saveDDL(agent)
			if err != nil {
				return err
			}
		}

		c.showJSON("Resulting Action", action)

		return nil
	}

	fmt.Println(`
An action is a hosted piece of logic that can be called remotely and
it takes input arguments and produce output data.

For example a package management agent would have actions like install,
uninstall, status and more.

Every agent can have as many actions as you want, we'll now prompt
for them until you are satisfied you added them all.

          Name: The name of the action, like "install"
   Description: A short 1 liner describing the purpose of the action
       Display: A hint to client tools about when to show the data,
                when interacting with 1000 nodes it's easy to miss
                the one that had a failure, setting this to "failed"
                will tell UIs to only show ones that failed.

                Likewise "ok" for only successful ones and "always" to
                show all results.
	`)

	for {
		fmt.Println()

		if len(agent.ActionNames()) > 0 {
			fmt.Printf("Existing Actions: %s\n\n", strings.Join(agent.ActionNames(), ", "))
		}

		if !c.AskBool("Add an action?") {
			break
		}

		err := addAction()
		if err != nil {
			return err
		}

		err = c.saveDDL(agent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Generator) askMetaData(agent *ddl.DDL) error {
	fmt.Println(`
First we need to gather meta data about the agent such as it's author, version and more
this metadata is used to keep an internal inventory of all the available services.

         Name: The name the agent would be reachable as, example package, acme_util
  Description: A short 1 liner description of the agent
       Author: Contact details to reach the author
      Version: Version in SemVer format
      License: The license to use, typically one in https://spdx.org/licenses/
          URL: A URL one can visit for further information about the agent
      Timeout: Maximum time in seconds any action will be allowed to run
     Provider: The provider to use - ruby for traditional mcollective ones,
               external for ones complying to the External Agent structure
      Service: Indicates an agent will be a service, hosted in a load sharing
               group rather than 1:n as normal agents.\n`)

	qs := []*survey.Question{
		c.askBasicItem("name", "Agent Name", "", survey.ToLower, c.shortnameValidator),
		c.askBasicItem("description", "Description", "", nil, survey.Required),
		c.askBasicItem("author", "Author", "", nil, survey.Required),
		c.askBasicItem("version", "Version", "", survey.ToLower, c.semVerValidator),
		c.askBasicItem("license", "License", "", nil, survey.Required),
		c.askBasicItem("url", "URL", "", survey.ToLower, c.urlValidator),
		c.askBasicItem("timeout", "Timeout", "", nil, survey.Required),
		c.askEnum("provider", "Backend Provider", "", []string{"ruby", "external", "golang"}, nil),
		c.askBasicItem("service", "Service", "", nil, c.boolValidator),
	}

	err := survey.Ask(qs, agent.Metadata)
	if err != nil {
		return err
	}

	c.showJSON("Resulting metadata", agent.Metadata)

	return nil
}
