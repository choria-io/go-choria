// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/lifecycle"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/data/ddl"
	"github.com/choria-io/go-choria/statistics"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/aagent"
)

// Agent is a generic choria agent
type Agent interface {
	Metadata() *Metadata
	Name() string
	HandleMessage(context.Context, inter.Message, protocol.Request, inter.ConnectorInfo, chan *AgentReply)
	SetServerInfo(ServerInfoSource)
	ServerInfo() ServerInfoSource
	ShouldActivate() bool
}

// ServerInfoSource provides data about a running server instance
type ServerInfoSource interface {
	AgentMetadata(string) (Metadata, bool)
	BuildInfo() *build.Info
	Classes() []string
	ConfigFile() string
	ConnectedServer() string
	DataFuncMap() (ddl.FuncMap, error)
	Facts() json.RawMessage
	Identity() string
	KnownAgents() []string
	LastProcessedMessage() time.Time
	MachineTransition(name string, version string, path string, id string, transition string) error
	MachinesStatus() ([]aagent.MachineState, error)
	NewEvent(t lifecycle.Type, opts ...lifecycle.Option) error
	PrepareForShutdown() error
	Provisioning() bool
	StartTime() time.Time
	Stats() statistics.ServerStats
	UpTime() int64
}

// AgentReply is a generic reply from an agent
type AgentReply struct {
	Body    []byte
	Request protocol.Request
	Message inter.Message
	Error   error
}

// Metadata describes an agent at a high level and is required for any agent
type Metadata struct {
	License     string `json:"license"`
	Author      string `json:"author"`
	Timeout     int    `json:"timeout"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Provider    string `json:"provider,omitempty"`
	Service     bool   `json:"service,omitempty"`
}

// Manager manages agents, handles registration, dispatches requests etc
type Manager struct {
	agents       map[string]Agent
	subs         map[string][]string
	fw           inter.Framework
	log          *logrus.Entry
	mu           *sync.Mutex
	conn         inter.ConnectorInfo
	serverInfo   ServerInfoSource
	denylist     []string
	requests     chan inter.ConnectorMessage
	servicesOnly bool
}

// NewServices creates an agent manager restricted to service agents
func NewServices(requests chan inter.ConnectorMessage, fw inter.Framework, conn inter.ConnectorInfo, srv ServerInfoSource, log *logrus.Entry) *Manager {
	m := New(requests, fw, conn, srv, log)
	m.servicesOnly = true
	m.log = m.log.WithField("service_host", true)

	return m
}

// New creates a new Agent Manager
func New(requests chan inter.ConnectorMessage, fw inter.Framework, conn inter.ConnectorInfo, srv ServerInfoSource, log *logrus.Entry) *Manager {
	return &Manager{
		agents:     make(map[string]Agent),
		subs:       make(map[string][]string),
		fw:         fw,
		log:        log.WithFields(logrus.Fields{"subsystem": "agents"}),
		mu:         &sync.Mutex{},
		requests:   requests,
		conn:       conn,
		serverInfo: srv,
	}
}

// DenyAgent adds an agent to the list of agent names not allowed to start
func (a *Manager) DenyAgent(agent string) {
	a.denylist = append(a.denylist, agent)
}

// ReplaceAgent allows an agent manager to replace an agent that is already known, and subsscribed, with another instance to facilitate in-place upgrades
func (a *Manager) ReplaceAgent(name string, agent Agent) error {
	if name == "" {
		return fmt.Errorf("agent name is required")
	}

	err := a.validateAgent(agent)
	if err != nil {
		a.log.Warnf("Denying agent %q update: %v", agent.Name(), err)
		return fmt.Errorf("invalid agent: %w", err)
	}

	if !agent.ShouldActivate() {
		return fmt.Errorf("replacement agent is not activating due to activation checks")
	}

	md := agent.Metadata()

	a.mu.Lock()
	defer a.mu.Unlock()

	ca, found := a.agents[name]
	if !found {
		return fmt.Errorf("agent %q is not currently known", name)
	}

	if ca.Metadata().Service != md.Service {
		return fmt.Errorf("replacement agent cannot change service property")
	}

	a.log.Infof("Replacing agent %s of type %s with a new instance moving from version %s to %s", name, md.Name, md.Version, ca.Metadata().Version)

	agent.SetServerInfo(a.serverInfo)

	a.agents[name] = agent

	return nil
}

func (a *Manager) validateAgent(agent Agent) error {
	md := agent.Metadata()

	if md.Timeout < 1 {
		return fmt.Errorf("timeout < 1")
	}

	if md.Name == "" {
		return fmt.Errorf("invalid metadata")
	}

	return nil
}

// UnRegisterAgent attempts to remove interest in messages for an agent
//
// Each agent has a number of subscriptions (one per collective) so this can fail for some
// while working for others, in this case the agent is essentially in an unrecoverable state
// however the cases where unsubscribe will error are quite few in the nats client as its
// not being-connected dependant and we handle most errors correctly.
//
// So this function will try to unsubscribe but if it fails, it will continue and finally unload
// the agent, any stale subscriptions then will be dropped by the handlers so its ok. We will treat
// unsbuscribe errors as non terminal, only logging errors.
func (a *Manager) UnRegisterAgent(name string, conn inter.AgentConnector) error {
	if name == "" {
		return fmt.Errorf("agent name is required")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	_, found := a.agents[name]
	if !found {
		return fmt.Errorf("unknown agent")
	}

	err := a.unSubscribeAgent(name, conn)
	if err != nil {
		a.log.Errorf("Could not unsubscribe all interest for agent %v: %v", name, err)
	}

	delete(a.agents, name)
	delete(a.subs, name)

	return nil
}

// RegisterAgent connects a new agent to the server instance, subscribe to all its targets etc
func (a *Manager) RegisterAgent(ctx context.Context, name string, agent Agent, conn inter.AgentConnector) error {
	if name == "" {
		return fmt.Errorf("agent name is required")
	}

	err := a.validateAgent(agent)
	if err != nil {
		a.log.Warnf("Denying agent %q: %v", name, err)
		return fmt.Errorf("invalid agent: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.servicesOnly && !agent.Metadata().Service {
		a.log.Infof("Denying non Service Agent %s", name)
		return nil
	}

	if !agent.ShouldActivate() {
		a.log.Infof("Agent %s not activating due to ShouldActivate checks", name)
		return nil
	}

	if a.agentDenied(name) {
		a.log.Infof("Denying agent %s based on agent deny list", name)
		return nil
	}

	a.log.Infof("Registering new agent %s of type %s", name, agent.Metadata().Name)

	agent.SetServerInfo(a.serverInfo)

	if _, found := a.agents[name]; found {
		return fmt.Errorf("agent %s is already registered", name)
	}

	err = a.subscribeAgent(ctx, name, agent, conn)
	if err != nil {
		return fmt.Errorf("could not register agent %s: %s", name, err)
	}

	a.agents[name] = agent

	return nil
}

// KnownAgents retrieves a list of known agents
func (a *Manager) KnownAgents() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	known := make([]string, 0, len(a.agents))

	for agent := range a.agents {
		known = append(known, agent)
	}

	sort.Strings(known)

	return known
}

func (a *Manager) agentDenied(name string) bool {
	for _, n := range a.denylist {
		if n == name {
			return true
		}
	}

	return false
}

func (a *Manager) unSubscribeAgent(name string, conn inter.AgentConnector) error {
	subs, ok := a.subs[name]
	if !ok {
		return nil
	}

	for _, sub := range subs {
		err := conn.Unsubscribe(sub)
		if err != nil {
			return err
		}
	}

	delete(a.subs, name)

	return nil
}

// Subscribes an agent to all its targets on the connector.  Should any subscription fail
// all the preceding subscriptions for this agents is unsubscribed and an error returned.
// Errors during the unsub is just ignored because it's quite possible that they would fail
// too but this avoids problems of messages arriving we did not expect.
//
// In practice though this is something done during bootstrap and failure here should exit
// the whole instance, so it's probably not needed
func (a *Manager) subscribeAgent(ctx context.Context, name string, agent Agent, conn inter.AgentConnector) error {
	if _, found := a.subs[name]; found {
		return fmt.Errorf("could not subscribe agent %s, it's already subscribed", name)
	}

	a.subs[name] = []string{}

	for _, collective := range a.fw.Configuration().Collectives {
		var target string
		group := ""

		if agent.Metadata().Service {
			target = conn.ServiceBroadcastTarget(collective, name)
			group = name
			a.log.Infof("Subscribing service agent %s to %s in group %s", name, target, group)
		} else {
			target = conn.AgentBroadcastTarget(collective, name)
			a.log.Infof("Subscribing agent %s to %s", name, target)
		}

		subname := fmt.Sprintf("%s.%s", collective, name)

		err := conn.QueueSubscribe(ctx, subname, target, group, a.requests)
		if err != nil {
			a.log.Errorf("could not subscribe agent %s to %s, rewinding all subscriptions for this agent", name, target)
			for _, sub := range a.subs[name] {
				conn.Unsubscribe(sub)
			}

			return fmt.Errorf("subscription failed: %s", err)
		}

		a.subs[name] = append(a.subs[name], subname)
	}

	return nil
}

// Get retrieves an agent by name
func (a *Manager) Get(name string) (Agent, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	agent, found := a.agents[name]

	return agent, found
}

// Dispatch sends a request to a agent and wait for a reply
func (a *Manager) Dispatch(ctx context.Context, wg *sync.WaitGroup, replies chan *AgentReply, msg inter.Message, request protocol.Request) {
	defer wg.Done()

	agent, found := a.Get(msg.Agent())
	if !found {
		a.log.Errorf("Received a message for agent %s that does not exist, discarding", msg.Agent())
		return
	}

	result := make(chan *AgentReply)

	td := time.Duration(agent.Metadata().Timeout) * time.Second
	a.log.Debugf("Handling message %s with timeout %s", msg.RequestID(), td)

	timeout, cancel := context.WithTimeout(context.Background(), td)
	defer cancel()

	go agent.HandleMessage(timeout, msg, request, a.conn, result)

	select {
	case reply := <-result:
		replies <- reply
	case <-ctx.Done():
		replies <- &AgentReply{
			Message: msg,
			Request: request,
			Error:   fmt.Errorf("agent dispatcher for request %s exiting on interrupt", msg.RequestID()),
		}
	case <-timeout.Done():
		replies <- &AgentReply{
			Message: msg,
			Request: request,
			Error:   fmt.Errorf("agent dispatcher for request %s exiting on %ds timeout", msg.RequestID(), agent.Metadata().Timeout),
		}
	}
}

// Logger is the logger the manager prefers new agents derive from
func (a *Manager) Logger() *logrus.Entry {
	return a.log
}

// Choria provides an instance of the choria framework
func (a *Manager) Choria() inter.Framework {
	return a.fw
}
