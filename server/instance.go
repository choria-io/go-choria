package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-choria/server/discovery"
	"github.com/choria-io/go-choria/server/registration"

	log "github.com/sirupsen/logrus"
)

// Instance is an independent copy of Choria
type Instance struct {
	fw                 *choria.Framework
	connector          choria.InstanceConnector
	cfg                *config.Config
	log                *log.Entry
	registration       *registration.Manager
	agents             *agents.Manager
	discovery          *discovery.Manager
	provisioning       bool
	startTime          time.Time
	lastMsgProcessed   time.Time
	agentDenyList      []string
	lifecycleComponent string
	machines           *aagent.AAgent

	requests chan *choria.ConnectorMessage

	shutdown func()
	mu       *sync.Mutex
}

// NewInstance creates a new choria server instance
func NewInstance(fw *choria.Framework) (i *Instance, err error) {
	i = &Instance{
		fw:               fw,
		cfg:              fw.Configuration(),
		requests:         make(chan *choria.ConnectorMessage),
		mu:               &sync.Mutex{},
		startTime:        time.Now(),
		lastMsgProcessed: time.Unix(0, 0),
		agentDenyList:    []string{},
	}

	i.log = log.WithFields(log.Fields{"identity": i.cfg.Identity, "component": "server"})
	i.discovery = discovery.New(fw, i.log)

	return i, nil
}

// Logger creates a new logger instance
func (srv *Instance) Logger(component string) *log.Entry {
	return srv.fw.Logger(component)
}

// Shutdown signals to the server that it should shutdown
func (srv *Instance) Shutdown() error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	cancel := srv.shutdown

	if cancel == nil {
		return fmt.Errorf("server is not running")
	}

	cancel()

	return nil
}

func (srv *Instance) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	var sctx context.Context
	srv.mu.Lock()
	sctx, srv.shutdown = context.WithCancel(ctx)
	srv.mu.Unlock()

	err := srv.initialConnect(sctx)
	if err != nil {
		srv.log.Errorf("Initial Choria Broker connection failed: %s", err)
		return fmt.Errorf("initial Choria Broker connection failed: %s", err)
	}

	wg.Add(1)
	go srv.WriteServerStatus(sctx, wg)

	srv.agents = agents.New(srv.requests, srv.fw, srv.connector, srv, srv.log)
	srv.registration = registration.New(srv.fw, srv.connector, srv.log)

	for _, n := range srv.agentDenyList {
		srv.agents.DenyAgent(n)
	}

	wg.Add(1)
	err = srv.registration.Start(sctx, wg)
	if err != nil {
		srv.log.Errorf("Could not initialize registration: %s", err)
		srv.connector.Close()

		return fmt.Errorf("could not initialize registration: %s", err)
	}

	err = srv.setupAdditionalAgentProviders(sctx)
	if err != nil {
		srv.log.Errorf("Could not initialize initial additional agent providers: %s", err)
		srv.connector.Close()

		return fmt.Errorf("could not initialize initial additional agent providers: %s", err)
	}

	err = srv.setupAdditionalAgents(sctx)
	if err != nil {
		srv.log.Errorf("Could not initialize initial additional agents: %s", err)
		srv.connector.Close()

		return fmt.Errorf("could not initialize initial additional agents: %s", err)
	}

	err = srv.subscribeNode(sctx)
	if err != nil {
		srv.log.Errorf("Could not subscribe node: %s", err)
		srv.connector.Close()

		return fmt.Errorf("could not subscribe node: %s", err)
	}

	srv.publishStartupEvent()

	wg.Add(1)
	go srv.publishAliveEvents(sctx, wg)

	err = srv.StartMachine(sctx, wg)
	if err != nil {
		srv.log.Errorf("Could not start Choria Autonomous Agent host: %s", err)
	}

	wg.Add(1)
	go srv.processRequests(sctx, wg)

	return nil
}

// RegisterRegistrationProvider adds a new provider for registration data to the registration subsystem
func (srv *Instance) RegisterRegistrationProvider(ctx context.Context, wg *sync.WaitGroup, provider registration.RegistrationDataProvider) error {
	return srv.registration.RegisterProvider(ctx, wg, provider)
}

// RegisterAgent adds a new agent to the running instance
func (srv *Instance) RegisterAgent(ctx context.Context, name string, agent agents.Agent) error {
	return srv.agents.RegisterAgent(ctx, name, agent, srv.connector)
}

// DenyAgent prevents an agent from being loaded, if it was already loaded this has no effect
func (srv *Instance) DenyAgent(agent string) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.agentDenyList = append(srv.agentDenyList, agent)
}
