// generated code; DO NOT EDIT

package scoutclient

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/protocol"
	rpcclient "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
)

// requester is a generic request handler
type requester struct {
	client *ScoutClient
	action string
	args   map[string]interface{}
}

// do performs the request
func (r *requester) do(ctx context.Context, handler func(pr protocol.Reply, r *rpcclient.RPCReply)) (*rpcclient.Stats, error) {
	targets := make([]string, 0)
	var err error

	r.client.Lock()
	copy(targets, r.client.targets)
	discoverer := r.client.ns
	filters := r.client.filters
	fw := r.client.fw

	opts := []rpcclient.RequestOption{rpcclient.Targets(targets)}
	opts = append(opts, r.client.clientRPCOpts...)
	r.client.Unlock()

	if len(targets) == 0 {
		r.client.infof("Starting discovery")
		targets, err = discoverer.Discover(ctx, fw, filters)
		if err != nil {
			return nil, err
		}

		if len(targets) == 0 {
			return nil, fmt.Errorf("no nodes were discovered")
		}
		r.client.infof("Discovered %d nodes", len(targets))
	}

	agent, err := rpcclient.New(r.client.fw, r.client.ddl.Metadata.Name, rpcclient.DDL(r.client.ddl))
	if err != nil {
		return nil, fmt.Errorf("could not create client: %s", err)
	}

	opts = append(opts, rpcclient.ReplyHandler(handler))

	r.client.infof("Invoking %s#%s action with %#v", r.client.ddl.Metadata.Name, r.action, r.args)

	res, err := agent.Do(ctx, r.action, r.args, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not perform disable request: %s", err)
	}

	return res.Stats(), nil
}
