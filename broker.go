package testutil

import (
	"fmt"
	"time"

	natsd "github.com/nats-io/nats-server/v2/server"
)

type Broker struct {
	NatsServer *natsd.Server
}

// ClientURL provides access the url to access the broker
func (b *Broker) ClientURL() string {
	if b.NatsServer ==nil {
		return ""
	}

	if b.NatsServer.Addr() == nil {
		return ""
	}

	return b.NatsServer.ClientURL()
}

func (b *Broker) Start() (err error) {
	if b.NatsServer!=nil {
		return fmt.Errorf("broker already exist, cannot start again")
	}

	b.NatsServer, err = natsd.NewServer(&natsd.Options{Port: -1})
	if err != nil {
		return  err
	}

	go b.NatsServer.Start()
	if !b.NatsServer.ReadyForConnections(10 * time.Second) {
		return fmt.Errorf("nats server did not start in 10 seconds")
	}

	return nil
}

// Stop shuts down the broker
func (b *Broker) Stop() {
	b.NatsServer.Shutdown()
	b.NatsServer =nil
}
