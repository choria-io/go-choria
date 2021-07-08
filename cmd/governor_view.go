package cmd

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/governor"
	"github.com/olekukonko/tablewriter"
)

type tGovViewCommand struct {
	command
	name string
}

func (g *tGovViewCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("view", "View the status of a Governor").Alias("show").Alias("v").Alias("s")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
	}

	return nil
}

func (g *tGovViewCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovViewCommand) Run(wg *sync.WaitGroup) (err error) {
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

	fmt.Printf("Configuration for Governor %s\n", gov.Name())
	fmt.Println()
	fmt.Printf("       Capacity: %d\n", gov.Limit())
	fmt.Printf("        Expires: %v\n", gov.MaxAge())
	fmt.Printf("       Replicas: %d\n", gov.Replicas())

	nfo, err := gov.Stream().Information()
	if err != nil {
		return err
	}

	fmt.Printf("  Active Leases: %d\n", nfo.State.Msgs)
	fmt.Println()

	if nfo.State.Msgs > 0 {
		fmt.Println()
		table := tablewriter.NewWriter(os.Stdout)
		defer table.Render()

		table.SetHeader([]string{"ID", "Process Name", "Age"})

		sub, err := conn.Nats().SubscribeSync(choria.Inbox(cfg.MainCollective, cfg.Identity))
		if err != nil {
			return err
		}
		defer sub.Unsubscribe()

		_, err = gov.Stream().NewConsumer(jsm.DeliverySubject(sub.Subject), jsm.DeliverAllAvailable(), jsm.AcknowledgeNone())
		if err != nil {
			return err
		}

		for {
			msg, err := sub.NextMsg(time.Second)
			if err != nil {
				return err
			}

			meta, err := jsm.ParseJSMsgMetadata(msg)
			if err != nil {
				continue
			}

			table.Append([]string{strconv.Itoa(int(meta.StreamSequence())), string(msg.Data), fmt.Sprintf("%v", time.Since(meta.TimeStamp()).Round(time.Millisecond))})
			if meta.Pending() == 0 {
				break
			}
		}
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGovViewCommand{})
}
