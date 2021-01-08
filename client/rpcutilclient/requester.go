// generated code; DO NOT EDIT

package rpcutilclient

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/protocol"
	rpcclient "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/gosuri/uiprogress"
)

// requester is a generic request handler
type requester struct {
	client   *RpcutilClient
	action   string
	args     map[string]interface{}
	progress *uiprogress.Bar
}

// do performs the request
func (r *requester) do(ctx context.Context, handler func(pr protocol.Reply, r *rpcclient.RPCReply)) (*rpcclient.Stats, error) {
	targets := make([]string, len(r.client.targets))
	var err error

	r.client.Lock()
	copy(targets, r.client.targets)
	discoverer := r.client.ns
	filters := r.client.filters
	fw := r.client.fw
	progress := r.client.clientOpts.progress

	if len(targets) == 0 {
		if progress {
			fmt.Print("Discovering nodes .... ")
		} else {
			r.client.infof("Starting discovery")
		}

		targets, err = discoverer.Discover(ctx, fw, filters)
		if err != nil {
			return nil, err
		}

		if len(targets) == 0 {
			return nil, fmt.Errorf("no nodes were discovered")
		}

		if progress {
			fmt.Printf("%d\n", len(targets))
		} else {
			r.client.infof("Discovered %d nodes", len(targets))
		}
	}

	opts := []rpcclient.RequestOption{rpcclient.Targets(targets)}
	opts = append(opts, r.client.clientRPCOpts...)
	if r.client.workers > 0 {
		opts = append(opts, rpcclient.Workers(r.client.workers))
	}
	r.client.Unlock()

	if progress {
		fmt.Println()
		r.configureProgress(len(targets))
	}

	agent, err := rpcclient.New(r.client.fw, r.client.ddl.Metadata.Name, rpcclient.DDL(r.client.ddl))
	if err != nil {
		return nil, fmt.Errorf("could not create client: %s", err)
	}

	if progress {
		opts = append(opts, rpcclient.ReplyHandler(func(pr protocol.Reply, rpcr *rpcclient.RPCReply) {
			r.progress.Incr()
			handler(pr, rpcr)
		}))
	} else {
		opts = append(opts, rpcclient.ReplyHandler(handler))
	}

	if !progress {
		r.client.infof("Invoking %s#%s action with %#v", r.client.ddl.Metadata.Name, r.action, r.args)
	}

	res, err := agent.Do(ctx, r.action, r.args, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not perform disable request: %s", err)
	}

	if progress {
		uiprogress.Stop()
		fmt.Println()
	}

	return res.Stats(), nil
}

func (r *requester) configureProgress(count int) {
	if !r.client.clientOpts.progress {
		return
	}

	r.progress = uiprogress.AddBar(count).AppendCompleted().PrependElapsed()
	r.progress.PrependFunc(func(b *uiprogress.Bar) string {
		if b.Current() < count {
			return r.client.fw.Colorize("red", "%d / %d", b.Current(), count)
		}

		return r.client.fw.Colorize("green", "%d / %d", b.Current(), count)
	})

	uiprogress.Start()
}
