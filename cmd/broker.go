package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/broker/network"
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
	net := choria.Config.Choria.BrokerNetwork
	discovery := choria.Config.Choria.BrokerDiscovery
	federation := choria.Config.Choria.BrokerFederation

	if !net && !discovery && !federation {
		return fmt.Errorf("All broker features are disabled")
	}

	if net {
		if err = r.runBroker(); err != nil {
			return fmt.Errorf("Starting the network broker failed: %s", err.Error())
		}
	}

	if federation {
		log.Warn("The Broker is configured to support Federation but it's not been implemented yet.")
	}

	if discovery {
		log.Warn("The Broker is configured to support Discovery but it's not been implemented yet.")
	}

	return
}

func (r *brokerRunCommand) runBroker() (err error) {
	r.server, err = network.NewServer(choria, debug)
	if err != nil {
		return fmt.Errorf("Could not set up Choria Network Broker: %s", err.Error())
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go r.server.Start(&wg)

	wg.Wait()

	return
}

func init() {
	cli.commands = append(cli.commands, &brokerCommand{})
	cli.commands = append(cli.commands, &brokerRunCommand{})
}
