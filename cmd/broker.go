package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/broker/federation"
	"github.com/choria-io/go-choria/broker/network"
	log "github.com/sirupsen/logrus"
)

type brokerCommand struct {
	command
}

type brokerRunCommand struct {
	command

	server     *network.Server
	federation *federation.FederationBroker
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
	wg := &sync.WaitGroup{}

	if !net && !discovery && !federation {
		return fmt.Errorf("All broker features are disabled")
	}

	if net {
		log.Info("Starting Network Broker")
		if err = r.runBroker(wg); err != nil {
			return fmt.Errorf("Starting the network broker failed: %s", err.Error())
		}
	}

	if federation {
		log.Info("Starting Federation Broker")
		if err = r.runFederation(wg); err != nil {
			return fmt.Errorf("Starting the federation broker failed: %s", err.Error())
		}
	}

	if discovery {
		log.Warn("The Broker is configured to support Discovery but it's not been implemented yet.")
	}

	wg.Wait()

	return
}

func (r *brokerRunCommand) runFederation(wg *sync.WaitGroup) (err error) {
	r.federation, err = federation.NewFederationBroker(choria.Config.Choria.FederationCluster, "1", choria)
	if err != nil {
		return fmt.Errorf("Could not set up Choria Federation Broker: %s", err.Error())
	}

	wg.Add(1)
	r.federation.Start(wg)

	return
}

func (r *brokerRunCommand) runBroker(wg *sync.WaitGroup) (err error) {
	r.server, err = network.NewServer(choria, debug)
	if err != nil {
		return fmt.Errorf("Could not set up Choria Network Broker: %s", err.Error())
	}

	wg.Add(1)
	go r.server.Start(wg)

	return
}

func init() {
	cli.commands = append(cli.commands, &brokerCommand{})
	cli.commands = append(cli.commands, &brokerRunCommand{})
}
