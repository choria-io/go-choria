package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/choria-io/go-choria/lifecycle"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/data/ddl"
	"github.com/choria-io/go-choria/statistics"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/aagent"
	"github.com/choria-io/go-choria/choria"
)

// Agent is a generic choria agent
type Agent interface {
	Metadata() *Metadata
	Name() string
	HandleMessage(context.Context, *choria.Message, protocol.Request, choria.ConnectorInfo, chan *AgentReply)
	SetServerInfo(ServerInfoSource)
	ServerInfo() ServerInfoSource
	ShouldActivate() bool
}

// ServerInfoSource provides data about a running server instance
type ServerInfoSource interface {
	AgentMetadata(string) (Metadata, bool)
	Classes() []string
	ConfigFile() string
	ConnectedServer() string
	Facts() json.RawMessage
	Identity() string
	KnownAgents() []string
	LastProcessedMessage() time.Time
	MachineTransition(name string, version string, path string, id string, transition string) error
	MachinesStatus() ([]aagent.MachineState, error)
	NewEvent(t lifecycle.Type, opts ...lifecycle.Option) error
	Provisioning() bool
	StartTime() time.Time
	Stats() statistics.ServerStats
	UpTime() int64
	PrepareForShutdown() error
	DataFuncMap() (ddl.FuncMap, error)
}

// AgentReply is a generic reply from an agent
type AgentReply struct {
	Body    []byte
	Request protocol.Request
	Message *choria.Message
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
}

// Manager manages agents, handles registration, dispatches requests etc
type Manager struct {
	agents     map[string]Agent
	subs       map[string][]string
	fw         *choria.Framework
	log        *logrus.Entry
	mu         *sync.Mutex
	conn       choria.ConnectorInfo
	serverInfo ServerInfoSource
	denylist   []string
	requests   chan *choria.ConnectorMessage
}

// New creates a new Agent Manager
func New(requests chan *choria.ConnectorMessage, fw *choria.Framework, conn choria.ConnectorInfo, srv ServerInfoSource, log *logrus.Entry) *Manager {
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

// RegisterAgent connects a new agent to the server instance, subscribe to all its targets etc
func (a *Manager) RegisterAgent(ctx context.Context, name string, agent Agent, conn choria.AgentConnector) error {
	a.mu.Lock()
	defer a.mu.Unlock()

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

	err := a.subscribeAgent(ctx, name, agent, conn)
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

func (a *Manager) agentDenied(agent string) bool {
	for _, n := range a.denylist {
		if n == agent {
			return true
		}
	}

	return false
}

// Subscribes an agent to all its targets on the connector.  Should any subscription fail
// all the preceding subscriptions for this agents is unsubscribed and an error returned.
// Errors during the unsub is just ignored because it's quite possible that they would fail
// too but this avoids problems of messages arriving we did not expect.
//
// In practice though this is something done during bootstrap and failure here should exit
// the whole instance, so it's probably not needed
func (a *Manager) subscribeAgent(ctx context.Context, name string, agent Agent, conn choria.AgentConnector) error {
	if _, found := a.subs[name]; found {
		return fmt.Errorf("could not subscribe agent %s, it's already subscribed", name)
	}

	a.subs[name] = []string{}

	for _, collective := range a.fw.Config.Collectives {
		target := conn.AgentBroadcastTarget(collective, name)
		subname := fmt.Sprintf("%s.%s", collective, name)

		a.log.Infof("Subscribing agent %s to %s", name, target)
		err := conn.QueueSubscribe(ctx, subname, target, "", a.requests)
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
func (a *Manager) Dispatch(ctx context.Context, wg *sync.WaitGroup, replies chan *AgentReply, msg *choria.Message, request protocol.Request) {
	defer wg.Done()

	agent, found := a.Get(msg.Agent)
	if !found {
		a.log.Errorf("Received a message for agent %s that does not exist, discarding", msg.Agent)
		return
	}

	result := make(chan *AgentReply)

	td := time.Duration(agent.Metadata().Timeout) * time.Second
	a.log.Debugf("Handling message %s with timeout %#v", msg.RequestID, td)

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
			Error:   fmt.Errorf("Agent dispatcher for request %s exiting on interrupt", msg.RequestID),
		}
	case <-timeout.Done():
		replies <- &AgentReply{
			Message: msg,
			Request: request,
			Error:   fmt.Errorf("Agent dispatcher for request %s exiting on %ds timeout", msg.RequestID, agent.Metadata().Timeout),
		}
	}
}

// Logger is the logger the manager prefers new agents derive from
func (a *Manager) Logger() *logrus.Entry {
	return a.log
}

// Choria provides an instance of the choria framework
func (a *Manager) Choria() *choria.Framework {
	return a.fw
}
