// generated code; DO NOT EDIT

package choria_registryclient

import (
	"fmt"
	"sync"
	"time"

	"context"

	coreclient "github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
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
	OverrideDiscoveryTime(start time.Time, end time.Time)
	UniqueRequestID() string
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
	NewMessage(payload string, agent string, collective string, msgType string, request inter.Message) (msg inter.Message, err error)
	NewReplyFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Reply, err error)
	NewTransportFromJSON(data string) (message protocol.TransportMessage, err error)
	MiddlewareServers() (servers srvcache.Servers, err error)
	NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *logrus.Entry) (conn inter.Connector, err error)
	NewRequestID() (string, error)
	Certname() string
	PQLQueryCertNames(query string) ([]string, error)
	Colorize(c string, format string, a ...interface{}) string
	ProgressWidth() int
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

// ChoriaRegistryClient to the choria_registry agent
type ChoriaRegistryClient struct {
	fw            ChoriaFramework
	cfg           *config.Config
	ddl           *agent.DDL
	ns            NodeSource
	clientOpts    *initOptions
	clientRPCOpts []rpcclient.RequestOption
	filters       []FilterFunc
	targets       []string
	workers       int
	exprFilter    string
	noReplies     bool

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
func Must(fw ChoriaFramework, opts ...InitializationOption) (client *ChoriaRegistryClient) {
	c, err := New(fw, opts...)
	if err != nil {
		panic(err)
	}

	return c
}

// New creates a new client to the choria_registry agent
func New(fw ChoriaFramework, opts ...InitializationOption) (client *ChoriaRegistryClient, err error) {
	c := &ChoriaRegistryClient{
		fw:            fw,
		ddl:           &agent.DDL{},
		clientRPCOpts: []rpcclient.RequestOption{},
		filters: []FilterFunc{
			FilterFunc(coreclient.AgentFilter("choria_registry")),
		},
		clientOpts: &initOptions{
			cfgFile: coreclient.UserConfig(),
		},
		targets: []string{},
	}

	for _, opt := range opts {
		opt(c.clientOpts)
	}

	c.cfg = c.fw.Configuration()

	if c.clientOpts.dt > 0 {
		c.cfg.DiscoveryTimeout = int(c.clientOpts.dt.Seconds())
	}

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
		c.clientOpts.logger = c.fw.Logger("choria_registry")
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
func (p *ChoriaRegistryClient) AgentMetadata() *Metadata {
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
func (p *ChoriaRegistryClient) DiscoverNodes(ctx context.Context) (nodes []string, err error) {
	p.Lock()
	defer p.Unlock()

	return p.ns.Discover(ctx, p.fw, p.filters)
}

// Ddl performs the ddl action
//
// Description: Retrieve the DDL for a specific plugin
//
// Required Inputs:
//    - name (string) - The name of the plugin
//    - plugin_type (string) - The type of plugin
//
// Optional Inputs:
//    - format (string) - The result format the plugin should be retrieved in
func (p *ChoriaRegistryClient) Ddl(inputName string, inputPluginType string) *DdlRequester {
	d := &DdlRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"name":        inputName,
				"plugin_type": inputPluginType,
			},
			action: "ddl",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}
