package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/governor"
)

type tGovAddCommand struct {
	command
	name     string
	limit    int64
	expire   time.Duration
	replicas int
}

func (g *tGovAddCommand) Setup() (err error) {
	if gen, ok := cmdWithFullCommand("tool governor"); ok {
		g.cmd = gen.Cmd().Command("add", "Adds or update a Governor")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
		g.cmd.Arg("capacity", "How many concurrent lease entries to allow").Required().Int64Var(&g.limit)
		g.cmd.Arg("expire", "Expire entries from the Governor after a period").Required().DurationVar(&g.expire)
		g.cmd.Arg("replicas", "Create a replicated Governor with this many replicas").Default("1").IntVar(&g.replicas)
	}

	return nil
}

func (g *tGovAddCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovAddCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("governor manager: %s", g.name), c.Logger("governor"))
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	gov, err := governor.NewJSGovernorManager(g.name, uint64(g.limit), g.expire, uint(g.replicas), mgr, false, governor.WithSubject(c.GovernorSubject(g.name)))
	if err != nil {
		return err
	}

	if gov.MaxAge() != g.expire || gov.Limit() != g.limit || gov.Replicas() != g.replicas {
		fmt.Println("Existing configuration:")
		fmt.Println()
		fmt.Printf("  Capacity: %d desired: %d\n", gov.Limit(), g.limit)
		fmt.Printf("   Expires: %v desired: %v\n", gov.MaxAge(), g.expire)
		fmt.Printf("  Replicas: %d desired: %d\n", gov.Replicas(), g.replicas)

		if gov.Replicas() != g.replicas {
			fmt.Println()
			fmt.Println("WARNING: replicas can not be updated")
			fmt.Println()
		}

		ans := false
		err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Update configuration for %s?", g.name),
			Default: ans,
		}, &ans)
		if err != nil {
			return err
		}

		if ans {
			err = gov.SetMaxAge(g.expire)
			if err != nil {
				return err
			}

			err = gov.SetLimit(uint64(g.limit))
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("Configuration:")
	fmt.Println()
	fmt.Printf("  Capacity: %d\n", gov.Limit())
	fmt.Printf("   Expires: %v\n", gov.MaxAge())
	fmt.Printf("  Replicas: %d\n", gov.Replicas())
	fmt.Println()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGovAddCommand{})
}
