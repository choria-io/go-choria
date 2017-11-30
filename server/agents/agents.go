package agents

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/sirupsen/logrus"
)

type Agent interface {
	Metadata() *Metadata
	Name() string
	Handle(*choria.Message) (*[]byte, error)
}

type Metadata struct {
	License     string
	Author      string
	Timeout     int
	Name        string
	Version     string
	URL         string
	Description string
}

type Manager struct {
	agents map[string]Agent
	subs   map[string][]string
	fw     *choria.Framework
	log    *logrus.Entry
	mu     *sync.Mutex
}

func New(fw *choria.Framework, log *logrus.Entry) *Manager {
	return &Manager{
		agents: make(map[string]Agent),
		subs:   make(map[string][]string),
		fw:     fw,
		log:    log.WithFields(logrus.Fields{"subsystem": "agents"}),
		mu:     &sync.Mutex{},
	}
}

func (a *Manager) RegisterAgent(name string, agent Agent, conn choria.AgentConnector) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.log.Infof("Registering new agent %s of type %s", name, agent.Metadata().Name)

	if _, found := a.agents[name]; found {
		return fmt.Errorf("Agent %s is already registered", name)
	}

	err := a.subscribeAgent(name, agent, conn)
	if err != nil {
		return fmt.Errorf("Could not register agent %s: %s", name, err.Error())
	}

	a.agents[name] = agent

	return nil
}

// Subscribes an agent to all its targets on the connector.  Should any subscription fail
// all the preceding subscriptions for this agents is unsubscribes and an error returned.
// Errors during the unsub is just ignored because it's quite possible that they would fail
// too but this avoids problems of messages arriving we did not expect.
//
// In practise though this is something done during bootstrap and failure here should exit
// the whole instance, so it's probably not needed
func (a *Manager) subscribeAgent(name string, agent Agent, conn choria.AgentConnector) error {
	if _, found := a.subs[name]; found {
		return fmt.Errorf("Could not subscribe agent %s, it's already subscribed", name)
	}

	a.subs[name] = []string{}

	for _, collective := range a.fw.Config.Collectives {
		target := conn.AgentBroadcastTarget(collective, name)
		subname := fmt.Sprintf("%s.%s", collective, name)

		a.log.Infof("Subscribing agent %s to %s", name, target)
		err := conn.Subscribe(subname, target, "")
		if err != nil {
			a.log.Errorf("Could not subscribe agent %s to %s, rewinding all subscriptions for this agent", name, target)
			for _, sub := range a.subs[name] {
				conn.Unsubscribe(sub)
			}

			return fmt.Errorf("Subscription failed: %s", err.Error())
		}

		a.subs[name] = append(a.subs[name], subname)
	}

	return nil
}

func (a *Manager) Get(name string) (Agent, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	agent, found := a.agents[name]

	return agent, found
}
