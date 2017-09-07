package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
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

	disableTls       bool
	disableTlsVerify bool
	pidFile          string

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
		r.cmd.Flag("disable-tls", "Disables TLS").Hidden().Default("false").BoolVar(&r.disableTls)
		r.cmd.Flag("disable-ssl-verification", "Disables SSL Verification").Hidden().Default("false").BoolVar(&r.disableTlsVerify)
		r.cmd.Flag("pid", "Write running PID to a file").StringVar(&r.pidFile)
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

	if r.pidFile != "" {
		err := ioutil.WriteFile(r.pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
		if err != nil {
			return fmt.Errorf("Could not write PID: %s", err.Error())
		}
	}

	if r.disableTls {
		choria.Config.DisableTLS = true
		log.Warn("Running with TLS disabled, not compatible with production use Choria.")
	}

	if r.disableTlsVerify {
		choria.Config.DisableTLSVerify = true
		log.Warn("Running with TLS Verification disabled, not compatible with production use Choria.")
	}

	if net {
		log.Info("Starting Network Broker")
		if err = r.runBroker(wg); err != nil {
			return fmt.Errorf("Starting the network broker failed: %s", err.Error())
		}
	}

	if federation {
		log.Infof("Starting Federation Broker on cluster %s", choria.Config.Choria.FederationCluster)
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
	r.federation, err = federation.NewFederationBroker(choria.Config.Choria.FederationCluster, choria)
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
