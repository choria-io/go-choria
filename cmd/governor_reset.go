package cmd

import (
	"fmt"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/governor"
)

type tGovResetCommand struct {
	command
	name  string
	force bool
}

func (g *tGovResetCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("reset", "Evicts all workers")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
		g.cmd.Flag("force", "Reset without prompting").Short('f').BoolVar(&g.force)
	}

	return nil
}

func (g *tGovResetCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovResetCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("governor manager: %s", g.name), c.Logger("governor"))
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	gov, err := governor.NewJSGovernorManager(g.name, 0, 0, 1, mgr, false)
	if err != nil {
		return err
	}

	entries, err := gov.Active()
	if err != nil {
		return err
	}

	if entries == 0 {
		fmt.Println("No lease entries to remove")
		return nil
	}

	if !g.force {
		ans := false
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Reset %s with %d lease entries?", g.name, entries),
			Default: ans,
		}, &ans)
		if err != nil {
			return err
		}
		if !ans {
			return nil
		}
	}

	return gov.Reset()
}

func init() {
	cli.commands = append(cli.commands, &tGovResetCommand{})
}
