package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/generators/client"
)

type tGenerateClientCommand struct {
	command

	ddlFile     string
	outDir      string
	packageName string
}

func (c *tGenerateClientCommand) Setup() (err error) {
	if gen, ok := cmdWithFullCommand("tool generate"); ok {
		c.cmd = gen.Cmd().Command("client", "Generate client bindings for an agent")
		c.cmd.Arg("ddl", "DDL file to use as input for the client").Required().ExistingFileVar(&c.ddlFile)
		c.cmd.Arg("target", "Directory to create the package in").Required().ExistingDirVar(&c.outDir)
		c.cmd.Flag("package", "Custom name for the generated package").StringVar(&c.packageName)
	}

	return nil
}

func (c *tGenerateClientCommand) Configure() (err error) {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (c *tGenerateClientCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	g := &client.Generator{
		DDLFile:     c.ddlFile,
		OutDir:      c.outDir,
		PackageName: c.packageName,
	}

	return g.GenerateClient()
}

func init() {
	cli.commands = append(cli.commands, &tGenerateClientCommand{})
}
