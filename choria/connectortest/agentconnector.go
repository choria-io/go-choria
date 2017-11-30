package connectortest

import (
	"fmt"
	"sync"
)

type AgentConnector struct {
	Subscribes   [][3]string
	Unsubscribes []string
	ActibeSubs   map[string]string
	NextErr      []error

	mu *sync.Mutex
}

func (a *AgentConnector) Init() {
	a.Subscribes = [][3]string{}
	a.Unsubscribes = []string{}
	a.ActibeSubs = make(map[string]string)
	a.mu = &sync.Mutex{}
	a.NextErr = []error{}
}

func (a *AgentConnector) Subscribe(name string, subject string, group string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.Subscribes = append(a.Subscribes, [3]string{name, subject, group})

	if err := a.nexterr(); err != nil {
		return err
	}

	if _, found := a.ActibeSubs[name]; found {
		return fmt.Errorf("%s already subscribed", name)
	}

	a.ActibeSubs[name] = subject

	return nil
}

func (a *AgentConnector) Unsubscribe(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, found := a.ActibeSubs[name]; !found {
		return fmt.Errorf("%s is not subscribed", name)
	}

	delete(a.ActibeSubs, name)

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
