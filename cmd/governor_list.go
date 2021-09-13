package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/nats-io/jsm.go"
)

type tGovListCommand struct {
	command
	json bool
}

func (g *tGovListCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("list", "Lists governors").Alias("ls")
		g.cmd.Flag("json", "Produce JSON output").Short('j').BoolVar(&g.json)
	}

	return nil
}

func (g *tGovListCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovListCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("governor manager: %s", "governor_list"), c.Logger("governor"))
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	known, err := mgr.StreamNames(&jsm.StreamNamesFilter{
		Subject: c.GovernorSubject("*"),
	})
	if err != nil {
		return err
	}

	for i := 0; i < len(known); i++ {
		known[i] = strings.TrimPrefix(known[i], "GOVERNOR_")
	}

	if g.json {
		out, err := json.MarshalIndent(known, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	if len(known) == 0 {
		fmt.Println("No Governors found")
		return nil
	}

	fmt.Println("Known Governors:")
	fmt.Println()

	for _, n := range known {
		fmt.Printf("\t%s\n", n)
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGovListCommand{})
}
