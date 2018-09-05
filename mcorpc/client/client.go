package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/choria-io/go-client/discovery/broadcast"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/srvcache"
	cclient "github.com/choria-io/go-client/client"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	addl "github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"

	"github.com/sirupsen/logrus"
)

// RPC is a MCollective compatible RPC client
type RPC struct {
	fw   *choria.Framework
	opts *RequestOptions
	log  *logrus.Entry

	agent string

	mu *sync.Mutex

	ddl *addl.DDL

	// used for testing only
	cl ChoriaClient
}

// RPCRequest is a basic RPC request
type RPCRequest struct {
	Agent  string          `json:"agent"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// RPCReply is a basic RPC reply
type RPCReply struct {
	Statuscode mcorpc.StatusCode `json:"statuscode"`
	Statusmsg  string            `json:"statusmsg"`
	Data       json.RawMessage   `json:"data"`
}

// RequestResult is the result of a request
type RequestResult interface {
	Stats() *Stats
}

// Handler is a function that should handle each reply synchronously
type Handler func(protocol.Reply, *RPCReply)

// ChoriaClient implements the connection to the Choria network
type ChoriaClient interface {
	Request(ctx context.Context, msg *choria.Message, handler cclient.Handler) (err error)
}

// Connector is a connection to the choria network
type Connector interface {
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *choria.ConnectorMessage) error
	Publish(msg *choria.Message) error
}

// Option configures the RPC client
type Option func(r *RPC)

// DDL supplies a DDL when creating the client thus avoiding a disk search
func DDL(d *addl.DDL) Option {
	return func(r *RPC) {
		r.ddl = d
	}
}

// New creates a new RPC request
//
// A DDL is required when one is not given using the DDL() option as argument
// attempts will be made to find it on the file system should this fail an error will be returned
func New(fw *choria.Framework, agent string, opts ...Option) (rpc *RPC, err error) {
	rpc = &RPC{
		fw:    fw,
		mu:    &sync.Mutex{},
		log:   fw.Logger("mcorpc"),
		agent: agent,
	}

	for _, opt := range opts {
		opt(rpc)
	}

	if rpc.ddl == nil {
		rpc.ddl, err = addl.Find(agent, fw.Config.LibDir)
		if err != nil {
			return nil, fmt.Errorf("could not load %s DDL: %s", agent, err)
		}
	}

	if rpc.ddl.Metadata.Name != agent {
		return nil, fmt.Errorf("the DDL does not describe the %s agent", agent)
	}

	return rpc, nil
}

func (r *RPC) setOptions(opts ...RequestOption) {
	r.opts = NewRequestOptions(r.fw, r.ddl)

	for _, opt := range opts {
		opt(r.opts)
	}
}

// Do performs a RPC request and optionally processes replies
//
// If a filter is supplied using the Filter() option and Targets() are not then discovery will be done for you
// using the broadcast method, should no nodes be discovered an error will be returned
func (r *RPC) Do(ctx context.Context, action string, payload interface{}, opts ...RequestOption) (RequestResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// we want to force the passing of options on every request
	r.setOptions(opts...)

	dctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if r.opts.Filter != nil && len(r.opts.Targets) == 0 {
		err := r.discover(ctx)
		if err != nil {
			return nil, fmt.Errorf("discovery failed: %s", err)
		}
	}

	msg, cl, err := r.setupMessage(dctx, action, payload, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not configure message: %s", err)
	}

	r.opts.totalStats.Start()
	defer r.opts.totalStats.End()

	ctr := 0

	// the client is always batched, when batched mode is not request the size of
	// the batch matches the size of the total targets and during setupMessage()
	// an appropriate connection will be made
	err = InGroups(r.opts.Targets, r.opts.BatchSize, func(nodes []string) error {
		r.opts.stats = NewStats()
		r.opts.stats.SetDiscoveredNodes(nodes)
		msg.DiscoveredHosts = nodes

		r.opts.stats.Start()
		defer r.opts.totalStats.Merge(r.opts.stats)
		defer r.opts.stats.End()

		if ctr > 0 {
			err := InterruptableSleep(dctx, r.opts.BatchSleep)
			if err != nil {
				return err
			}
		}

		r.log.Debugf("Performing batched request %d for %d/%d nodes", ctr, len(nodes), len(r.opts.Targets))

		err = r.request(dctx, msg, cl)
		if err != nil {
			return fmt.Errorf("could not create request: %s", err)
		}

		ctr++

		return nil
	})

	r.opts.totalStats.SetAction(action)
	r.opts.totalStats.SetAgent(r.agent)

	return &RequestOptions{totalStats: r.opts.totalStats}, err
}

func (r *RPC) discover(ctx context.Context) error {
	b := broadcast.New(r.fw)

	r.opts.totalStats.StartDiscover()
	defer r.opts.totalStats.EndDiscover()

	n, err := b.Discover(ctx, broadcast.Filter(r.opts.Filter), broadcast.Timeout(r.opts.DiscoveryTimeout), broadcast.Name(r.opts.ConnectionName), broadcast.Collective(r.opts.Collective))
	if err != nil {
		return err
	}

	if len(n) == 0 {
		return fmt.Errorf("no targets were discovered")
	}

	r.opts.Targets = n
	r.opts.totalStats.SetDiscoveredNodes(n)

	return nil
}

func (r *RPC) setupMessage(ctx context.Context, action string, payload interface{}, opts ...RequestOption) (msg *choria.Message, cl ChoriaClient, err error) {
	pj, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("could not encode payload: %s", err)
	}

	rpcreq := &RPCRequest{
		Agent:  r.agent,
		Action: action,
		Data:   pj,
	}

	rpcp, err := json.Marshal(rpcreq)
	if err != nil {
		return nil, nil, fmt.Errorf("could not encode request: %s", err)
	}

	msg, err = r.fw.NewMessage(string(rpcp), r.agent, r.fw.Config.MainCollective, "request", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create Message: %s", err)
	}

	err = r.opts.ConfigureMessage(msg)
	if err != nil {
		return nil, nil, fmt.Errorf("could not configure Message: %s", err)
	}

	cl = r.cl

	if r.cl == nil {
		if r.opts.BatchSize == len(r.opts.Targets) || r.opts.ProcessReplies == false {
			cl, err = r.unbatchedClient()
			if err != nil {
				return nil, nil, err
			}
		} else {
			cl, err = r.batchedClient(ctx, msg.RequestID)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return msg, cl, err
}

func (r *RPC) unbatchedClient() (cl ChoriaClient, err error) {
	cl, err = cclient.New(
		r.fw,
		cclient.Receivers(r.opts.Workers),
		cclient.Timeout(r.opts.Timeout),
		cclient.OnPublishStart(r.opts.stats.StartPublish),
		cclient.OnPublishFinish(r.opts.stats.EndPublish),
		cclient.Name(r.opts.ConnectionName),
	)
	if err != nil {
		return nil, fmt.Errorf("could not setup client: %s", err)
	}

	return cl, nil
}

func (r *RPC) batchedClient(ctx context.Context, msgid string) (cl ChoriaClient, err error) {
	conn, err := r.connectBatchedConnection(ctx, fmt.Sprintf("%s_batched", msgid))
	if err != nil {
		return nil, fmt.Errorf("could not connect batched network connection: %s", err)
	}

	cl, err = cclient.New(
		r.fw,
		cclient.Receivers(r.opts.Workers),
		cclient.Timeout(r.opts.Timeout),
		cclient.OnPublishStart(r.opts.stats.StartPublish),
		cclient.OnPublishFinish(r.opts.stats.EndPublish),
		cclient.Connection(conn),
		cclient.Name(r.opts.ConnectionName),
	)
	if err != nil {
		return nil, fmt.Errorf("could not set up batched client: %s", err)
	}

	return cl, nil
}

// Reset removes the cached options, any further Do() calls need to specify full options
func (r *RPC) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.opts = nil
	r.cl = nil
}

func (r *RPC) request(ctx context.Context, msg *choria.Message, cl ChoriaClient) error {
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := cl.Request(rctx, msg, r.handlerFactory(rctx, cancel))
	if err != nil {
		return fmt.Errorf("could not create request: %s", err)
	}

	return nil
}

func (r *RPC) handlerFactory(ctx context.Context, cancel func()) cclient.Handler {
	if !r.opts.ProcessReplies {
		return nil
	}

	handler := func(ctx context.Context, rawmsg *choria.ConnectorMessage) {
		reply, err := r.fw.NewReplyFromTransportJSON(rawmsg.Data, false)
		if err != nil {
			r.opts.stats.FailedRequestInc()
			r.log.Errorf("Could not process a reply: %s", err)
			return
		}

		r.opts.stats.RecordReceived(reply.SenderID())

		rpcreply, err := ParseReplyData([]byte(reply.Message()))
		if err != nil {
			r.opts.stats.FailedRequestInc()
			r.log.Errorf("Could not process reply from %s: %s", reply.SenderID(), err)
			return
		}

		if rpcreply.Statuscode == mcorpc.OK {
			r.opts.stats.PassedRequestInc()
		} else {
			r.opts.stats.FailedRequestInc()
		}

		if r.opts.Handler != nil {
			r.opts.Handler(reply, rpcreply)
		}

		if r.opts.stats.All() {
			cancel()
			return
		}
	}

	return handler
}

func (r *RPC) connectBatchedConnection(ctx context.Context, name string) (Connector, error) {
	servers := func() ([]srvcache.Server, error) {
		return r.fw.MiddlewareServers()
	}

	connector, err := r.fw.NewConnector(ctx, servers, name, r.log)
	if err != nil {
		return nil, fmt.Errorf("could not create connector: %s", err)
	}

	closer := func() {
		select {
		case <-ctx.Done():
			r.log.Debugf("Closing batched connection %s", name)
			connector.Close()
		}
	}

	go closer()

	return connector, nil
}
