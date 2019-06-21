package connectortest

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/choria"
	nats "github.com/nats-io/nats.go"
)

type AgentConnector struct {
	Subscribes   [][3]string
	Unsubscribes []string
	ActiveSubs   map[string]string
	NextErr      []error

	Input chan *nats.Msg

	mu *sync.Mutex
}

func (a *AgentConnector) Init() {
	a.Subscribes = [][3]string{}
	a.Unsubscribes = []string{}
	a.ActiveSubs = make(map[string]string)
	a.mu = &sync.Mutex{}
	a.NextErr = []error{}
	a.Input = make(chan *nats.Msg)
}

func (a *AgentConnector) QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *choria.ConnectorMessage) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.Subscribes = append(a.Subscribes, [3]string{name, subject, group})

	if err := a.nexterr(); err != nil {
		return err
	}

	if _, found := a.ActiveSubs[name]; found {
		return fmt.Errorf("%s already subscribed", name)
	}

	a.ActiveSubs[name] = subject

	return nil
}

func (a *AgentConnector) Unsubscribe(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, found := a.ActiveSubs[name]; !found {
		return fmt.Errorf("%s is not subscribed", name)
	}

	delete(a.ActiveSubs, name)

	a.Unsubscribes = append(a.Unsubscribes, name)

	return nil
}

func (a *AgentConnector) AgentBroadcastTarget(collective string, agent string) string {
	return fmt.Sprintf("%s.broadcast.agent.%s", collective, agent)
}

func (a *AgentConnector) nexterr() error {
	var err error

	if len(a.NextErr) > 0 {
		err = a.NextErr[0]

		if len(a.NextErr) == 1 {
			a.NextErr = []error{}
		} else {
			a.NextErr = append(a.NextErr[:0], a.NextErr[1:]...)
		}
	}

	return err
}

func (a *AgentConnector) ConnectionOptions() nats.Options {
	return nats.Options{}
}

func (a *AgentConnector) ConnectionStats() nats.Statistics {
	return nats.Statistics{}
}

func (a *AgentConnector) ConnectedServer() string {
	return "nats://nats.example.net:4222"
}
