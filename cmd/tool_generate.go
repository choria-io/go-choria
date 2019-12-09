package cmd

import "sync"

type tGenerateCommand struct {
	command
}

func (g *tGenerateCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		g.cmd = tool.Cmd().Command("generate", "Generates choria related data")
	}

	return nil
}

func (g *tGenerateCommand) Configure() error {
	return nil
}

func (g *tGenerateCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGenerateCommand{})
}
