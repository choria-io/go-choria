// generated code; DO NOT EDIT

package choria_utilclient

import (
	"fmt"
	"sync"
	"time"

	"context"

	"github.com/choria-io/go-choria/choria"
	coreclient "github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
	rpcclient "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/sirupsen/logrus"
)

// Stats are the statistics for a request
type Stats interface {
	Agent() string
	Action() string
	All() bool
	NoResponseFrom() []string
	UnexpectedResponseFrom() []string
	DiscoveredCount() int
	DiscoveredNodes() *[]string
	FailCount() int
	OKCount() int
	ResponsesCount() int
	PublishDuration() (time.Duration, error)
	RequestDuration() (time.Duration, error)
	DiscoveryDuration() (time.Duration, error)
}

// NodeSource discovers nodes
type NodeSource interface {
	Reset()
	Discover(ctx context.Context, fw ChoriaFramework, filters []FilterFunc) ([]string, error)
}

// ChoriaFramework is the Choria framework
type ChoriaFramework interface {
	Logger(string) *logrus.Entry
	SetLogger(*logrus.Logger)
	Configuration() *config.Config
	NewMessage(payload string, agent string, collective string, msgType string, request *choria.Message) (msg *choria.Message, err error)
	NewReplyFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Reply, err error)
	NewTransportFromJSON(data string) (message protocol.TransportMessage, err error)
	MiddlewareServers() (servers srvcache.Servers, err error)
	NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *logrus.Entry) (conn choria.Connector, err error)
	NewRequestID() (string, error)
	Certname() string
	PQLQueryCertNames(query string) ([]string, error)
	Colorize(c string, format string, a ...interface{}) string
}

// FilterFunc can generate a Choria filter
type FilterFunc func(f *protocol.Filter) error

// RenderFormat is the format used by the RenderResults helper
type RenderFormat int

const (
	// JSONFormat renders the results as a JSON document
	JSONFormat RenderFormat = iota

	// TextFormat renders the results as a Choria typical result set in line with choria req output
	TextFormat

	// TableFormat renders all successful responses in a table
	TableFormat

	// TXTFooter renders only the request summary statistics
	TXTFooter
)

// DisplayMode overrides the DDL display hints
type DisplayMode uint8

const (
	// DisplayDDL shows results based on the configuration in the DDL file
	DisplayDDL = DisplayMode(iota)
	// DisplayOK shows only passing results
	DisplayOK
	// DisplayFailed shows only failed results
	DisplayFailed
	// DisplayAll shows all results
	DisplayAll
	// DisplayNone shows no results
	DisplayNone
)

type Log interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})
}

// ChoriaUtilClient to the choria_util agent
type ChoriaUtilClient struct {
	fw            ChoriaFramework
	cfg           *config.Config
	ddl           *agent.DDL
	ns            NodeSource
	clientOpts    *initOptions
	clientRPCOpts []rpcclient.RequestOption
	filters       []FilterFunc
	targets       []string
	workers       int

	sync.Mutex
}

// Metadata is the agent metadata
type Metadata struct {
	License     string `json:"license"`
	Author      string `json:"author"`
	Timeout     int    `json:"timeout"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// Must create a new client and panics on error
func Must(opts ...InitializationOption) (client *ChoriaUtilClient) {
	c, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return c
}

// New creates a new client to the choria_util agent
func New(opts ...InitializationOption) (client *ChoriaUtilClient, err error) {
	c := &ChoriaUtilClient{
		ddl:           &agent.DDL{},
		clientRPCOpts: []rpcclient.RequestOption{},
		filters: []FilterFunc{
			FilterFunc(coreclient.AgentFilter("choria_util")),
		},
		clientOpts: &initOptions{
			cfgFile: choria.UserConfig(),
		},
		targets: []string{},
	}

	for _, opt := range opts {
		opt(c.clientOpts)
	}

	c.fw, err = choria.New(c.clientOpts.cfgFile)
	if err != nil {
		return nil, fmt.Errorf("could not initialize Choria: %s", err)
	}

	c.cfg = c.fw.Configuration()

	if c.clientOpts.ns == nil {
		switch c.cfg.DefaultDiscoveryMethod {
		case "choria":
			c.clientOpts.ns = &PuppetDBNS{}
		default:
			c.clientOpts.ns = &BroadcastNS{}
		}
	}
	c.ns = c.clientOpts.ns

	if c.clientOpts.logger == nil {
		c.clientOpts.logger = c.fw.Logger("choria_util")
	} else {
		c.fw.SetLogger(c.clientOpts.logger.Logger)
	}

	c.ddl, err = DDL()
	if err != nil {
		return nil, fmt.Errorf("could not parse embedded DDL: %s", err)
	}

	return c, nil
}

// AgentMetadata is the agent metadata this client supports
func (p *ChoriaUtilClient) AgentMetadata() *Metadata {
	return &Metadata{
		License:     p.ddl.Metadata.License,
		Author:      p.ddl.Metadata.Author,
		Timeout:     p.ddl.Metadata.Timeout,
		Name:        p.ddl.Metadata.Name,
		Version:     p.ddl.Metadata.Version,
		URL:         p.ddl.Metadata.URL,
		Description: p.ddl.Metadata.Description,
	}
}

// DiscoverNodes performs a discovery using the configured filter and node source
func (p *ChoriaUtilClient) DiscoverNodes(ctx context.Context) (nodes []string, err error) {
	p.Lock()
	defer p.Unlock()

	return p.ns.Discover(ctx, p.fw, p.filters)
}

// Info performs the info action
//
// Description: Choria related information from the running Daemon and Middleware
func (p *ChoriaUtilClient) Info() *InfoRequester {
	d := &InfoRequester{
		outc: nil,
		r: &requester{
			args:   map[string]interface{}{},
			action: "info",
			client: p,
		},
	}

	return d
}

// MachineStates performs the machine_states action
//
// Description: States of the hosted Choria Autonomous Agents
func (p *ChoriaUtilClient) MachineStates() *MachineStatesRequester {
	d := &MachineStatesRequester{
		outc: nil,
		r: &requester{
			args:   map[string]interface{}{},
			action: "machine_states",
			client: p,
		},
	}

	return d
}

// MachineTransition performs the machine_transition action
//
// Description: Attempts to force a transition in a hosted Choria Autonomous Agent
//
// Required Inputs:
//    - transition (string) - The transition event to send to the machine
//
// Optional Inputs:
//    - instance (string) - Machine Instance ID
//    - name (string) - Machine Name
//    - path (string) - Machine Path
//    - version (string) - Machine Version
func (p *ChoriaUtilClient) MachineTransition(transitionI string) *MachineTransitionRequester {
	d := &MachineTransitionRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"transition": transitionI,
			},
			action: "machine_transition",
			client: p,
		},
	}

	return d
}
