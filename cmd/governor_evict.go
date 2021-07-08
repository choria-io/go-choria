package cmd

import (
	"fmt"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/lifecycle"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/governor"
)

type tGovEvictCommand struct {
	command
	name  string
	seq   uint64
	force bool
}

func (g *tGovEvictCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("evict", "Evicts a specific worker")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
		g.cmd.Arg("worker", "The lease ID to remove").Required().Uint64Var(&g.seq)
		g.cmd.Flag("force", "Reset without prompting").Short('f').BoolVar(&g.force)
	}

	return nil
}

func (g *tGovEvictCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovEvictCommand) Run(wg *sync.WaitGroup) (err error) {
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
			Message: fmt.Sprintf("Evict lease %d from %s?", g.seq, g.name),
			Default: ans,
		}, &ans)
		if err != nil {
			return err
		}
		if !ans {
			return nil
		}
	}

	name, err := gov.Evict(g.seq)
	if err != nil {
		return err
	}

	event, err := lifecycle.New(lifecycle.Governor, lifecycle.Identity(name), lifecycle.Component("CLI"), lifecycle.GovernorSequence(g.seq), lifecycle.GovernorName(g.name), lifecycle.GovernorType(lifecycle.GovernorEvictEvent))
	if err == nil {
		lifecycle.PublishEvent(event, conn)
		conn.Close()
	}

	fmt.Printf("Evicted %q from slot %d\n", name, g.seq)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGovEvictCommand{})
}
