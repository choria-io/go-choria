// generated code; DO NOT EDIT

package executorclient

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
	Discover(ctx context.Context, fw inter.Framework, filters []FilterFunc) ([]string, error)
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
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Panicf(format string, args ...any)
}

// ExecutorClient to the executor agent
type ExecutorClient struct {
	fw            inter.Framework
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
func Must(fw inter.Framework, opts ...InitializationOption) (client *ExecutorClient) {
	c, err := New(fw, opts...)
	if err != nil {
		panic(err)
	}

	return c
}

// New creates a new client to the executor agent
func New(fw inter.Framework, opts ...InitializationOption) (client *ExecutorClient, err error) {
	c := &ExecutorClient{
		fw:            fw,
		ddl:           &agent.DDL{},
		clientRPCOpts: []rpcclient.RequestOption{},
		filters: []FilterFunc{
			FilterFunc(coreclient.AgentFilter("executor")),
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
		c.clientOpts.logger = c.fw.Logger("executor")
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
func (p *ExecutorClient) AgentMetadata() *Metadata {
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
func (p *ExecutorClient) DiscoverNodes(ctx context.Context) (nodes []string, err error) {
	p.Lock()
	defer p.Unlock()

	return p.ns.Discover(ctx, p.fw, p.filters)
}

// Signal performs the signal action
//
// Description: Sends a signal to a process
//
// Required Inputs:
//   - id (string) - The unique ID for the job
//   - signal (int64) - The signal to send
func (p *ExecutorClient) Signal(inputId string, inputSignal int64) *SignalRequester {
	d := &SignalRequester{
		outc: nil,
		r: &requester{
			args: map[string]any{
				"id":     inputId,
				"signal": inputSignal,
			},
			action: "signal",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// List performs the list action
//
// Description: Lists jobs matching certain criteria
//
// Optional Inputs:
//   - action (string) - The action that created a job
//   - agent (string) - The agent that create a job
//   - before (int64) - Unix timestamp to limit jobs on
//   - caller (string) - The caller id that created a job
//   - command (string) - The command that was executed
//   - completed (bool) - Limit to jobs that were completed
//   - identity (string) - The host identity that created the job
//   - requestid (string) - The Request ID that created the job
//   - running (bool) - Limits to running jobs
//   - since (int64) - Unix timestamp to limit jobs on
func (p *ExecutorClient) List() *ListRequester {
	d := &ListRequester{
		outc: nil,
		r: &requester{
			args:   map[string]any{},
			action: "list",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// Status performs the status action
//
// Description: Requests the status of a job by ID
//
// Required Inputs:
//   - id (string) - The unique ID for the job
func (p *ExecutorClient) Status(inputId string) *StatusRequester {
	d := &StatusRequester{
		outc: nil,
		r: &requester{
			args: map[string]any{
				"id": inputId,
			},
			action: "status",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}
