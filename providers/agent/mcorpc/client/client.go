// Copyright (c) 2018-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/inter"
	"github.com/expr-lang/expr/vm"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/discovery/broadcast"
	"github.com/choria-io/go-choria/providers/discovery/puppetdb"

	cclient "github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	addl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"

	"github.com/sirupsen/logrus"
)

// RPC is a MCollective compatible RPC client
type RPC struct {
	fw   inter.Framework
	opts *RequestOptions
	log  *logrus.Entry
	cfg  *config.Config
	dm   string

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

// RequestResult is the result of a request
type RequestResult interface {
	Stats() *Stats
}

// Handler is a function that should handle each reply synchronously
type Handler func(protocol.Reply, *RPCReply)

// ChoriaClient implements the connection to the Choria network
type ChoriaClient interface {
	Request(ctx context.Context, msg inter.Message, handler cclient.Handler) (err error)
}

// Connector is a connection to the choria network
type Connector interface {
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan inter.ConnectorMessage) error
	Publish(msg inter.Message) error
}

// Option configures the RPC client
type Option func(r *RPC)

// DDL supplies a DDL when creating the client thus avoiding a disk search
func DDL(d *addl.DDL) Option {
	return func(r *RPC) {
		r.ddl = d
	}
}

// DiscoveryMethod sets a specific discovery method
func DiscoveryMethod(dm string) Option {
	return func(r *RPC) {
		r.dm = dm
	}
}

// New creates a new RPC request
//
// A DDL is required when one is not given using the DDL() option as argument
// attempts will be made to find it on the file system should this fail an error will be returned
func New(fw inter.Framework, agent string, opts ...Option) (rpc *RPC, err error) {
	rpc = &RPC{
		fw:    fw,
		cfg:   fw.Configuration(),
		mu:    &sync.Mutex{},
		log:   fw.Logger("rpc"),
		agent: agent,
		dm:    fw.Configuration().DefaultDiscoveryMethod,
	}

	for _, opt := range opts {
		opt(rpc)
	}

	return rpc, nil
}

func (r *RPC) setOptions(opts ...RequestOption) (err error) {
	r.opts, err = NewRequestOptions(r.fw, r.ddl)
	if err != nil {
		return err
	}

	for _, opt := range opts {
		opt(r.opts)
	}

	if r.ddl.Metadata.Service {
		r.opts.Workers = 1
		r.opts.RequestType = inter.ServiceRequestMessageType
	}

	return nil
}

func (r *RPC) ResolveDDL(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ddl != nil {
		return nil
	}

	ddl, err := r.resolveDDL(ctx)
	if err != nil {
		return err
	}

	r.ddl = ddl

	return nil
}

func (r *RPC) resolveDDL(ctx context.Context) (*addl.DDL, error) {
	if r.ddl != nil {
		return r.ddl, nil
	}

	resolvers, err := r.fw.DDLResolvers()
	if err != nil {
		return nil, err
	}

	for _, resolver := range resolvers {
		r.log.Debugf("Attempting to resolve agent DDL using %s", resolver)

		data, err := resolver.DDLBytes(ctx, "agent", r.agent, r.fw)
		if err == nil {
			ddl, err := addl.NewFromBytes(data)
			if err != nil {
				return nil, err
			}
			r.ddl = ddl

			return ddl, nil
		} else {
			r.log.Debugf("DDL Resolution failed: %s", err)
		}
	}

	return nil, fmt.Errorf("could not resolve %s DDL in any resolver: %s", r.agent, err)
}

// DDL returns the active DDL for this client
func (r *RPC) DDL() *addl.DDL {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ddl
}

// Do perform a RPC request and optionally processes replies
//
// If a filter is supplied using the Filter() option and Targets() are not then discovery will be done for you
// using the broadcast method, should no nodes be discovered an error will be returned
func (r *RPC) Do(ctx context.Context, action string, payload any, opts ...RequestOption) (RequestResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.resolveDDL(ctx)
	if err != nil {
		return nil, err
	}

	if r.ddl.Metadata.Name != r.agent {
		return nil, fmt.Errorf("the DDL does not describe the %s agent", r.agent)
	}

	// we want to force the passing of options on every request
	err = r.setOptions(opts...)
	if err != nil {
		return nil, err
	}

	dctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if len(r.opts.Targets) == 0 && (r.opts.RequestType != inter.ServiceRequestMessageType || !r.ddl.Metadata.Service) {
		err = r.discover(ctx)
		if err != nil {
			return nil, fmt.Errorf("discovery failed: %s", err)
		}
	}

	discoveredCnt := len(r.opts.Targets)
	msg, cl, err := r.setupMessage(dctx, action, payload, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not configure message: %s", err)
	}

	if r.opts.DiscoveryEndCB != nil {
		err = r.opts.DiscoveryEndCB(discoveredCnt, len(r.opts.Targets))
		if err != nil {
			return nil, err
		}
	}

	r.opts.totalStats.Start()
	defer r.opts.totalStats.End()

	r.opts.totalStats.SetAction(action)
	r.opts.totalStats.SetAgent(r.agent)

	switch r.opts.RequestType {
	case inter.ServiceRequestMessageType:
		err = r.doServiceRequest(dctx, msg, cl)
	default:
		err = r.doBatchedRequest(ctx, msg, cl)
	}

	return &RequestOptions{totalStats: r.opts.totalStats}, err
}

func (r *RPC) doBatchedRequest(ctx context.Context, msg inter.Message, cl ChoriaClient) error {
	// the client is always batched, when batched mode is not request the size of
	// the batch matches the size of the total targets and during setupMessage()
	// an appropriate connection will be made

	ctr := 0

	return InGroups(r.opts.Targets, r.opts.BatchSize, func(nodes []string) error {
		stats := NewStats()
		stats.SetDiscoveredNodes(nodes)
		msg.SetDiscoveredHosts(nodes)

		stats.Start()
		defer func(s *Stats) {
			s.End()
			r.opts.totalStats.Merge(s)
		}(stats)

		if ctr > 0 {
			err := InterruptableSleep(ctx, r.opts.BatchSleep)
			if err != nil {
				return err
			}
		}

		r.log.Debugf("Performing batched request %d for %d/%d nodes", ctr, len(nodes), len(r.opts.Targets))

		err := r.request(ctx, msg, cl, stats)
		if err != nil {
			return err
		}

		ctr++

		return nil
	})
}

func (r *RPC) doServiceRequest(ctx context.Context, msg inter.Message, cl ChoriaClient) error {
	stats := NewStats()

	var responded []string
	handler := r.opts.Handler
	r.opts.Handler = func(r protocol.Reply, rpc *RPCReply) {
		responded = append(responded, r.SenderID())
		if handler != nil {
			handler(r, rpc)
		}
	}

	err := r.request(ctx, msg, cl, stats)
	if len(responded) > 0 {
		stats.SetDiscoveredNodes(responded)
		r.opts.totalStats.SetDiscoveredNodes(responded)
		stats.RecordReceived(responded[0])
	}

	stats.End()

	r.opts.totalStats.Merge(stats)

	return err
}

func (r *RPC) discover(ctx context.Context) error {
	if len(r.opts.Filter.Compound) > 0 {
		r.dm = "choria"
	}
	if r.opts.DiscoveryStartCB != nil {
		r.opts.DiscoveryStartCB()
	}

	r.opts.totalStats.StartDiscover()
	defer r.opts.totalStats.EndDiscover()

	if r.opts.Filter == nil {
		r.opts.Filter = protocol.NewFilter()
	}

	r.opts.Filter.AddAgentFilter(r.agent)

	var n []string
	var err error

	// TODO: other discovery options? honestly the magical discovery here should just go
	switch r.dm {
	case "choria":
		pdb := puppetdb.New(r.fw)
		n, err = pdb.Discover(ctx, puppetdb.Filter(r.opts.Filter), puppetdb.Timeout(r.opts.DiscoveryTimeout), puppetdb.Collective(r.opts.Collective))

	default:
		b := broadcast.New(r.fw)
		n, err = b.Discover(ctx, broadcast.Filter(r.opts.Filter), broadcast.Timeout(r.opts.DiscoveryTimeout), broadcast.Name(r.opts.ConnectionName), broadcast.Collective(r.opts.Collective))
	}
	if err != nil {
		return err
	}

	if len(n) == 0 {
		return fmt.Errorf("no targets were discovered")
	}

	r.opts.Targets = n

	return nil
}

func (r *RPC) setupMessage(ctx context.Context, action string, payload any, opts ...RequestOption) (msg inter.Message, cl ChoriaClient, err error) {
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

	msgType := inter.RequestMessageType
	if r.ddl.Metadata.Service {
		msgType = inter.ServiceRequestMessageType
		r.opts.Workers = 1
	}

	msg, err = r.fw.NewMessage(rpcp, r.agent, r.cfg.MainCollective, msgType, nil)
	if err != nil {
		return nil, nil, err
	}

	err = r.opts.ConfigureMessage(msg)
	if err != nil {
		return nil, nil, fmt.Errorf("could not configure Message: %s", err)
	}

	cl = r.cl

	if r.cl == nil {
		if r.opts.BatchSize == len(r.opts.Targets) || !r.opts.ProcessReplies {
			cl, err = r.unbatchedClient()
			if err != nil {
				return nil, nil, err
			}
		} else {
			cl, err = r.batchedClient(ctx, msg.RequestID())
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
		cclient.OnPublishStart(r.opts.totalStats.StartPublish),
		cclient.OnPublishFinish(r.opts.totalStats.EndPublish),
		cclient.Name(r.opts.ConnectionName),
	)
	if err != nil {
		return nil, fmt.Errorf("could not setup client: %s", err)
	}

	return cl, nil
}

func (r *RPC) batchedClient(ctx context.Context, msgid string) (cl ChoriaClient, err error) {
	conn, err := r.connectBatchedConnection(ctx, fmt.Sprintf("%s_%s_batched", r.opts.ConnectionName, msgid))
	if err != nil {
		return nil, fmt.Errorf("could not connect batched network connection: %s", err)
	}

	cl, err = cclient.New(
		r.fw,
		cclient.Receivers(r.opts.Workers),
		cclient.Timeout(r.opts.Timeout),
		cclient.OnPublishStart(r.opts.totalStats.StartPublish),
		cclient.OnPublishFinish(r.opts.totalStats.EndPublish),
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

func (r *RPC) request(ctx context.Context, msg inter.Message, cl ChoriaClient, stats *Stats) error {
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := cl.Request(rctx, msg, r.handlerFactory(rctx, cancel, stats))
	if err != nil {
		return err
	}

	return nil
}

func (r *RPC) handlerFactory(_ context.Context, cancel context.CancelFunc, stats *Stats) cclient.Handler {
	if !r.opts.ProcessReplies {
		return nil
	}

	var prog *vm.Program

	handler := func(ctx context.Context, rawmsg inter.ConnectorMessage) {
		reply, err := r.fw.NewReplyFromTransportJSON(rawmsg.Data(), false)
		if err != nil {
			stats.FailedRequestInc()
			r.log.Errorf("Could not process a reply: %s", err)
			return
		}

		// defer because we do not do any discovery so recording the response here would mark it as unknown
		if r.opts.RequestType != inter.ServiceRequestMessageType {
			stats.RecordReceived(reply.SenderID())
		}

		rpcreply, err := ParseReply(reply)
		switch {
		case err != nil:
			stats.FailedRequestInc()
			r.log.Errorf("Could not process reply from %s: %s", reply.SenderID(), err)
			return
		case rpcreply.Statuscode == mcorpc.OK:
			stats.PassedRequestInc()
		default:
			stats.FailedRequestInc()
		}

		if r.opts.Handler != nil {
			shouldShow := true
			if r.opts.ReplyExprFilter != "" {
				shouldShow, prog, err = rpcreply.MatchExpr(r.opts.ReplyExprFilter, prog)
				if err != nil {
					r.log.Errorf("Expr filter parsing failed in reply from %s: %s", reply.SenderID(), err)
				}
			}

			if shouldShow {
				r.opts.Handler(reply, rpcreply)
			} else {
				r.opts.Handler(reply, nil)
			}
		}

		if stats.All() {
			cancel()
			return
		}
	}

	return handler
}

func (r *RPC) connectBatchedConnection(ctx context.Context, name string) (Connector, error) {
	connector, err := r.fw.NewConnector(ctx, r.fw.MiddlewareServers, name, r.log)
	if err != nil {
		return nil, err
	}

	closer := func() {
		<-ctx.Done()

		r.log.Debugf("Closing batched connection %s", name)
		connector.Close()
		connector.Close()
	}

	go closer()

	return connector, nil
}
