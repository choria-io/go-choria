package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/jsm.go"
)

type kvLSCommand struct {
	command
}

func (k *kvLSCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("list", "List buckets").Alias("ls")
	}

	return nil
}

func (k *kvLSCommand) Configure() error {
	return commonConfigure()
}

func (k *kvLSCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, "kv manager", c.Logger("kv"))
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	found := 0
	table := util.NewMarkdownTable("Bucket", "History", "Values")

	mgr.EachStream(func(s *jsm.Stream) {
		if !strings.HasPrefix(s.Name(), "KV_") {
			return
		}

		parts := strings.SplitN(s.Name(), "_", 2)
		if len(parts) != 2 {
			return
		}

		state, err := s.LatestState()
		if err != nil {
			return
		}

		found++
		table.Append([]string{
			parts[1],
			strconv.Itoa(int(s.MaxMsgsPerSubject())),
			strconv.Itoa(int(state.Msgs)),
		})
	})

	if found == 0 {
		fmt.Println("No Key-Value stores found")
		return nil
	}

	table.Render()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvLSCommand{})
}
