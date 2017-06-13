package cmd

import (
	"fmt"

	"github.com/choria-io/go-choria/network"
	log "github.com/sirupsen/logrus"
)

type brokerCommand struct {
	command
}

type brokerRunCommand struct {
	command
	server *network.Server
}

// broker
func (b *brokerCommand) Setup() (err error) {
	b.cmd = cli.app.Command("broker", "Choria Network Broker")

	return
}

func (b *brokerCommand) Run() (err error) {
	return
}

// broker run
func (r *brokerRunCommand) Setup() (err error) {
	if broker, ok := cmdWithFullCommand("broker"); ok {
		r.cmd = broker.Cmd().Command("run", "Runs a Choria Network Broker instance").Default()
	}

	return
}

func (r *brokerRunCommand) Run() (err error) {
	r.server, err = network.NewServer(Choria, debug)
	if err != nil {
		return fmt.Errorf("Could not set up Choria Network Broker: %s", err.Error())
	}

	log.Debug("Starting goroutine for the NATS broker")

	go r.server.Start()

	return
}

func init() {
	cli.commands = append(cli.commands, &brokerCommand{})
	cli.commands = append(cli.commands, &brokerRunCommand{})
}
