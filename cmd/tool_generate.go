package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-config"
	ddl "github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
	"github.com/xeipuuv/gojsonschema"
)

type tGenerateCommand struct {
	command
	targetType   string
	jsonOut      string
	rubyOut      string
	skipVerify   bool
	forceConvert bool
}

func (g *tGenerateCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		g.cmd = tool.Cmd().Command("generate", "Generates choria related data")
		g.cmd.Arg("type", "The type of data to generate").Required().EnumVar(&g.targetType, "ddl")
		g.cmd.Arg("json_output", "Where to place the JSON output").Required().StringVar(&g.jsonOut)
		g.cmd.Arg("ruby_output", "Where to place the Ruby output").StringVar(&g.rubyOut)
		g.cmd.Flag("skip-verify", "Do not verify the JSON file against the DDL Schema").Default("false").BoolVar(&g.skipVerify)
		g.cmd.Flag("convert", "Convert JSON to DDL without prompting").Default("false").BoolVar(&g.forceConvert)
	}

	return nil
}

func (g *tGenerateCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("Could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (g *tGenerateCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	switch g.targetType {
	case "ddl":
		if g.jsonOut != "" && choria.FileExist(g.jsonOut) {
			if g.forceConvert || g.askBool(fmt.Sprintf("JSON ddl %s already exist, convert it to Ruby", g.jsonOut)) {
				return g.convertToRuby()
			}
		}

		err = g.generateDDL()
		if err != nil {
			return fmt.Errorf("ddl generation failed: %s", err)
		}

	default:
		return fmt.Errorf("generating %s data is not supported", g.targetType)
	}

	return nil
}

func (g *tGenerateCommand) ValidateJSON(agent *ddl.DDL) error {
	j, err := json.Marshal(agent)
	if err != nil {
		return err
	}

	sloader := gojsonschema.NewReferenceLoader("https://choria.io/schemas/mcorpc/ddl/v1/agent.json")
	dloader := gojsonschema.NewBytesLoader(j)

	result, err := gojsonschema.Validate(sloader, dloader)
	if err != nil {
		return fmt.Errorf("schema validation failed: %s", err)
	}

	if !result.Valid() {
		fmt.Printf("The generate DDL does not pass validation against https://choria.io/schemas/mcorpc/ddl/v1/agent.json:\n\n")
		for _, err := range result.Errors() {
			fmt.Printf(" - %s\n", err)
		}

		return fmt.Errorf("JSON DDL validation failed")
	}

	return nil
}

func (g *tGenerateCommand) convertToRuby() error {
	jddl, err := ddl.New(g.jsonOut)
	if err != nil {
		return err
	}

	if !g.skipVerify {
		fmt.Println("Validating JSON DDL against the schema...")
		err = g.ValidateJSON(jddl)
		if err != nil {
			fmt.Printf("\nWARN: DDL does not pass JSON Schema Validation: %s\n", err)
		}
		fmt.Println()
	}

	rddl, err := jddl.ToRuby()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(g.rubyOut, []byte(rddl), 0644)
}

func (g *tGenerateCommand) generateDDL() error {
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

	err := g.askMetaData(agent)
	if err != nil {
		return err
	}

	err = g.saveDDL(agent)
	if err != nil {
		return err
	}

	err = g.askActions(agent)
	if err != nil {
		return err
	}

	err = g.saveDDL(agent)
	if err != nil {
		return err
	}

	if !g.skipVerify {
		fmt.Println("Validating JSON DDL against the schema...")
		err = g.ValidateJSON(agent)
		if err != nil {
			fmt.Printf("WARN: DDL does not pass JSON Schema Validation: %s\n", err)
		}
		fmt.Println()
	}

	return nil
}

func (g *tGenerateCommand) saveDDL(agent *ddl.DDL) error {
	err := g.saveJSON(agent)
	if err != nil {
		return err
	}

	return g.saveRuby(agent)
}

func (g *tGenerateCommand) saveRuby(agent *ddl.DDL) error {
	if g.rubyOut == "" {
		return nil
	}

	out, err := os.Create(g.rubyOut)
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

func (g *tGenerateCommand) saveJSON(agent *ddl.DDL) error {
	out, err := os.Create(g.jsonOut)
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

func (g *tGenerateCommand) askBasicItem(name string, prompt string, help string, t survey.Transformer, v survey.Validator) *survey.Question {
	return &survey.Question{
		Name:      name,
		Prompt:    &survey.Input{Message: prompt, Help: help},
		Validate:  v,
		Transform: t,
	}
}

func (g *tGenerateCommand) askBool(m string) bool {
	should := false
	prompt := &survey.Confirm{
		Message: m,
	}
	survey.AskOne(prompt, &should)
	return should

}

func (g *tGenerateCommand) askEnum(name string, prompt string, help string, valid []string, v survey.Validator) *survey.Question {
	return &survey.Question{
		Name:     name,
		Prompt:   &survey.Select{Message: prompt, Help: help, Options: valid},
		Validate: v,
	}
}

func (g *tGenerateCommand) showJSON(m string, d interface{}) error {
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

func (g *tGenerateCommand) urlValidator(v interface{}) error {
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

func (g *tGenerateCommand) semVerValidator(v interface{}) error {
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

func (g *tGenerateCommand) shortnameValidator(v interface{}) error {
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

func (g *tGenerateCommand) askActions(agent *ddl.DDL) error {
	addAction := func() error {
		action := &ddl.Action{
			Input:       make(map[string]*ddl.ActionInputItem),
			Output:      make(map[string]*ddl.ActionOutputItem),
			Aggregation: []ddl.ActionAggregateItem{},
		}

		qs := []*survey.Question{
			g.askBasicItem("name", "Action Name", "", survey.ToLower, func(v interface{}) error {
				act := v.(string)

				if act == "" {
					return fmt.Errorf("an action name is required")
				}

				if agent.HaveAction(act) {
					return fmt.Errorf("already have an action %s", act)
				}

				return g.shortnameValidator(v)
			}),

			g.askBasicItem("description", "Description", "", nil, survey.Required),
			g.askEnum("display", "Display Hint", "", []string{"ok", "failed", "always"}, survey.Required),
		}

		err = survey.Ask(qs, action)
		if err != nil {
			return err
		}

		agent.Actions = append(agent.Actions, action)

		err = g.saveDDL(agent)
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

			if !g.askBool("Add an input?") {
				break
			}

			input := &ddl.ActionInputItem{}
			name := ""
			survey.AskOne(&survey.Input{Message: "Input Name:"}, &name, survey.WithValidator(survey.Required), survey.WithValidator(func(v interface{}) error {
				i := v.(string)
				if i == "" {
					return fmt.Errorf("input name is required")
				}

				_, ok := action.Input[i]
				if ok {
					return fmt.Errorf("input %s already exist", i)
				}

				return g.shortnameValidator(v)
			}))
			qs := []*survey.Question{
				g.askBasicItem("prompt", "Prompt", "", nil, survey.Required),
				g.askBasicItem("description", "Description", "", nil, survey.Required),
				g.askEnum("type", "Data Type", "", []string{"integer", "number", "float", "string", "boolean", "list", "hash", "array"}, survey.Required),
				g.askBasicItem("optional", "Optional (t/f)", "", nil, survey.Required),
			}

			err = survey.Ask(qs, input)
			if err != nil {
				return err
			}

			if input.Type == "string" {
				qs = []*survey.Question{
					g.askBasicItem("maxlength", "Max Length", "", nil, survey.Required),
					g.askEnum("validation", "Validation", "", []string{"shellsafe", "ipv4address", "ipv6address", "ipaddress", "regex"}, survey.Required),
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
					Message: "Valid Values (comma seperated)",
					Help:    "List of valid values for this input seperated by commas",
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
				input.Default, err = ddl.ValToDDLType(input.Type, deflt)
				if err != nil {
					return fmt.Errorf("default for %s does not validate: %s", name, err)
				}
			}

			action.Input[name] = input

			err = g.saveDDL(agent)
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

			if !g.askBool("Add an output?") {
				break
			}

			output := &ddl.ActionOutputItem{}
			name := ""
			survey.AskOne(&survey.Input{Message: "Name:"}, &name, survey.WithValidator(survey.Required), survey.WithValidator(func(v interface{}) error {
				i := v.(string)
				if i == "" {
					return fmt.Errorf("output name is required")
				}

				_, ok := action.Output[i]
				if ok {
					return fmt.Errorf("output %s already exist", i)
				}

				return g.shortnameValidator(v)
			}))
			qs := []*survey.Question{
				g.askBasicItem("description", "Description", "", nil, survey.Required),
				g.askEnum("type", "Data Type", "", []string{"integer", "number", "float", "string", "boolean", "list", "hash", "array"}, survey.Required),
				g.askBasicItem("displayas", "Display As", "", nil, survey.Required),
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
				output.Default, err = ddl.ValToDDLType(output.Type, deflt)
				if err != nil {
					return fmt.Errorf("default for %s does not validate: %s", name, err)
				}
			}

			action.Output[name] = output
			err = g.saveDDL(agent)
			if err != nil {
				return err
			}
		}

		g.showJSON("Resulting Action", action)

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

		if !g.askBool("Add an action?") {
			break
		}

		err = addAction()
		if err != nil {
			return err
		}

		err = g.saveDDL(agent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *tGenerateCommand) askMetaData(agent *ddl.DDL) error {
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
`)

	qs := []*survey.Question{
		g.askBasicItem("name", "Agent Name", "", survey.ToLower, g.shortnameValidator),
		g.askBasicItem("description", "Description", "", nil, survey.Required),
		g.askBasicItem("author", "Author", "", nil, survey.Required),
		g.askBasicItem("version", "Version", "", survey.ToLower, g.semVerValidator),
		g.askBasicItem("license", "License", "", nil, survey.Required),
		g.askBasicItem("url", "URL", "", survey.ToLower, g.urlValidator),
		g.askBasicItem("timeout", "Timeout", "", nil, survey.Required),
		g.askEnum("provider", "Backend Provider", "", []string{"ruby", "external", "golang"}, nil),
	}

	err = survey.Ask(qs, agent.Metadata)
	if err != nil {
		return err
	}

	g.showJSON("Resulting metadata", agent.Metadata)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGenerateCommand{})
}
