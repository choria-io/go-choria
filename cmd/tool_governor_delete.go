package cmd

import (
	"fmt"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/governor"
)

type tGovDeleteCommand struct {
	command
	name  string
	force bool
}

func (g *tGovDeleteCommand) Setup() (err error) {
	if gen, ok := cmdWithFullCommand("tool governor"); ok {
		g.cmd = gen.Cmd().Command("delete", "Deletes a Governor").Alias("rm")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
		g.cmd.Arg("force", "Reset without prompting").BoolVar(&g.force)
	}

	return nil
}

func (g *tGovDeleteCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovDeleteCommand) Run(wg *sync.WaitGroup) (err error) {
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

	if !g.force {
		ans := false
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Delete %s with %d active entries?", g.name, entries),
			Default: ans,
		}, &ans)
		if err != nil {
			return err
		}
		if !ans {
			return nil
		}
	}

	return gov.Stream().Delete()
}

func init() {
	cli.commands = append(cli.commands, &tGovDeleteCommand{})
}
