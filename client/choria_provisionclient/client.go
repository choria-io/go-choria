// generated code; DO NOT EDIT

package choria_provisionclient

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
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})
}

// ChoriaProvisionClient to the choria_provision agent
type ChoriaProvisionClient struct {
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
func Must(fw inter.Framework, opts ...InitializationOption) (client *ChoriaProvisionClient) {
	c, err := New(fw, opts...)
	if err != nil {
		panic(err)
	}

	return c
}

// New creates a new client to the choria_provision agent
func New(fw inter.Framework, opts ...InitializationOption) (client *ChoriaProvisionClient, err error) {
	c := &ChoriaProvisionClient{
		fw:            fw,
		ddl:           &agent.DDL{},
		clientRPCOpts: []rpcclient.RequestOption{},
		filters: []FilterFunc{
			FilterFunc(coreclient.AgentFilter("choria_provision")),
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
		c.clientOpts.logger = c.fw.Logger("choria_provision")
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
func (p *ChoriaProvisionClient) AgentMetadata() *Metadata {
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
func (p *ChoriaProvisionClient) DiscoverNodes(ctx context.Context) (nodes []string, err error) {
	p.Lock()
	defer p.Unlock()

	return p.ns.Discover(ctx, p.fw, p.filters)
}

// Configure performs the configure action
//
// Description: Configure the Choria Server
//
// Required Inputs:
//    - config (string) - The configuration to apply to this node
//
// Optional Inputs:
//    - action_policies (map[string]interface{}) - Map of Action Policy documents indexed by file name
//    - ca (string) - PEM text block for the CA
//    - certificate (string) - PEM text block for the certificate
//    - ecdh_public (string) - Required when sending a private key
//    - key (string) - A RSA private key
//    - opa_policies (map[string]interface{}) - Map of Open Policy Agent Policy documents indexed by file name
//    - server_jwt (string) - JWT file used to identify the server to the broker for ed25519 based authentication
//    - ssldir (string) - Directory for storing the certificate in
//    - token (string) - Authentication token to pass to the server
func (p *ChoriaProvisionClient) Configure(inputConfig string) *ConfigureRequester {
	d := &ConfigureRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"config": inputConfig,
			},
			action: "configure",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// Gen25519 performs the gen25519 action
//
// Description: Generates a new ED25519 keypair
//
// Required Inputs:
//    - nonce (string) - Single use token to be signed by the private key being generated
//    - token (string) - Authentication token to pass to the server
func (p *ChoriaProvisionClient) Gen25519(inputNonce string, inputToken string) *Gen25519Requester {
	d := &Gen25519Requester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"nonce": inputNonce,
				"token": inputToken,
			},
			action: "gen25519",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// Gencsr performs the gencsr action
//
// Description: Request a CSR from the Choria Server
//
// Required Inputs:
//    - token (string) - Authentication token to pass to the server
//
// Optional Inputs:
//    - C (string) - Country Code
//    - L (string) - Locality or municipality (such as city or town name)
//    - O (string) - Organization
//    - OU (string) - Organizational Unit
//    - ST (string) - State
//    - cn (string) - The certificate Common Name to place in the CSR
func (p *ChoriaProvisionClient) Gencsr(inputToken string) *GencsrRequester {
	d := &GencsrRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"token": inputToken,
			},
			action: "gencsr",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// ReleaseUpdate performs the release_update action
//
// Description: Performs an in-place binary update and restarts Choria
//
// Required Inputs:
//    - repository (string) - HTTP(S) server hosting the update repository
//    - token (string) - Authentication token to pass to the server
//    - version (string) - Package version to update to
func (p *ChoriaProvisionClient) ReleaseUpdate(inputRepository string, inputToken string, inputVersion string) *ReleaseUpdateRequester {
	d := &ReleaseUpdateRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"repository": inputRepository,
				"token":      inputToken,
				"version":    inputVersion,
			},
			action: "release_update",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// Jwt performs the jwt action
//
// Description: Re-enable provision mode in a running Choria Server
//
// Required Inputs:
//    - token (string) - Authentication token to pass to the server
func (p *ChoriaProvisionClient) Jwt(inputToken string) *JwtRequester {
	d := &JwtRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"token": inputToken,
			},
			action: "jwt",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// Reprovision performs the reprovision action
//
// Description: Reenable provision mode in a running Choria Server
//
// Required Inputs:
//    - token (string) - Authentication token to pass to the server
func (p *ChoriaProvisionClient) Reprovision(inputToken string) *ReprovisionRequester {
	d := &ReprovisionRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"token": inputToken,
			},
			action: "reprovision",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}

// Restart performs the restart action
//
// Description: Restart the Choria Server
//
// Required Inputs:
//    - token (string) - Authentication token to pass to the server
//
// Optional Inputs:
//    - splay (float64) - The configuration to apply to this node
func (p *ChoriaProvisionClient) Restart(inputToken string) *RestartRequester {
	d := &RestartRequester{
		outc: nil,
		r: &requester{
			args: map[string]interface{}{
				"token": inputToken,
			},
			action: "restart",
			client: p,
		},
	}

	action, _ := p.ddl.ActionInterface(d.r.action)
	action.SetDefaults(d.r.args)

	return d
}
