package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/generators/ddl"
)

type tGenerateDDLCommand struct {
	command

	jsonOut      string
	rubyOut      string
	skipVerify   bool
	forceConvert bool
}

func (c *tGenerateDDLCommand) Setup() (err error) {
	if gen, ok := cmdWithFullCommand("tool generate"); ok {
		c.cmd = gen.Cmd().Command("ddl", "Generate and convert DDL files")
		c.cmd.Arg("json_output", "Where to place the JSON output").Required().StringVar(&c.jsonOut)
		c.cmd.Arg("ruby_output", "Where to place the Ruby output").StringVar(&c.rubyOut)
		c.cmd.Flag("skip-verify", "Do not verify the JSON file against the DDL Schema").Default("false").BoolVar(&c.skipVerify)
		c.cmd.Flag("convert", "Convert JSON to DDL without prompting").Default("false").BoolVar(&c.forceConvert)
	}

	return nil
}

func (c *tGenerateDDLCommand) Configure() (err error) {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (c *tGenerateDDLCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	generator := &ddl.Generator{
		JSONOut:      c.jsonOut,
		RubyOut:      c.rubyOut,
		SkipVerify:   c.skipVerify,
		ForceConvert: c.forceConvert,
	}

	if c.jsonOut != "" && choria.FileExist(c.jsonOut) {
		if c.forceConvert || generator.AskBool(fmt.Sprintf("JSON ddl %s already exist, convert it to Ruby", c.jsonOut)) {
			return generator.ConvertToRuby()
		}
	}

	err = generator.GenerateDDL()
	if err != nil {
		return fmt.Errorf("ddl generation failed: %s", err)
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGenerateDDLCommand{})
}
