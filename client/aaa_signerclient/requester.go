// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// generated code; DO NOT EDIT

package aaa_signerclient

import (
	"context"
	"fmt"
	"time"

	"github.com/choria-io/go-choria/protocol"
	rpcclient "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/gosuri/uiprogress"
)

// requester is a generic request handler
type requester struct {
	client   *AaaSignerClient
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

	opts := []rpcclient.RequestOption{}
	discoveryStart := time.Now()

	if r.client.ddl.Metadata.Service {
		opts = append(opts, rpcclient.ServiceRequest(), rpcclient.Workers(1))
	} else if len(targets) == 0 {
		if r.client.clientOpts.progress {
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

		if r.client.clientOpts.progress {
			fmt.Printf("%d\n", len(targets))
		} else {
			r.client.infof("Discovered %d nodes", len(targets))
		}
	}
	discoveryEnd := time.Now()

	if r.client.workers > 0 {
		opts = append(opts, rpcclient.Workers(r.client.workers))
	}

	if r.client.exprFilter != "" {
		opts = append(opts, rpcclient.ReplyExprFilter(r.client.exprFilter))
	}

	if len(targets) > 0 {
		opts = append(opts, rpcclient.Targets(targets))
	}

	opts = append(opts, r.client.clientRPCOpts...)

	r.client.Unlock()

	if r.client.clientOpts.progress {
		fmt.Println()
		r.configureProgress(len(targets))
	}

	agent, err := rpcclient.New(r.client.fw, r.client.ddl.Metadata.Name, rpcclient.DDL(r.client.ddl))
	if err != nil {
		return nil, fmt.Errorf("could not create client: %s", err)
	}

	if r.client.clientOpts.progress {
		opts = append(opts, rpcclient.ReplyHandler(func(pr protocol.Reply, rpcr *rpcclient.RPCReply) {
			r.progress.Incr()
			handler(pr, rpcr)
		}))
	} else if !r.client.noReplies {
		opts = append(opts, rpcclient.ReplyHandler(handler))
	}

	if !r.client.clientOpts.progress {
		r.client.infof("Invoking %s#%s action with %#v", r.client.ddl.Metadata.Name, r.action, r.args)
	}

	res, err := agent.Do(ctx, r.action, r.args, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not perform request: %s", err)
	}

	if r.client.clientOpts.progress {
		uiprogress.Stop()
		fmt.Println()
	}

	if !discoveryStart.IsZero() && !discoveryEnd.IsZero() {
		res.Stats().OverrideDiscoveryTime(discoveryStart, discoveryEnd)
	}

	return res.Stats(), nil
}

func (r *requester) configureProgress(count int) {
	if !r.client.clientOpts.progress {
		return
	}

	r.progress = uiprogress.AddBar(count).AppendCompleted().PrependElapsed()

	width := r.client.fw.ProgressWidth()
	if width == -1 {
		r.client.clientOpts.progress = false
		return
	}

	r.progress.Width = width

	r.progress.PrependFunc(func(b *uiprogress.Bar) string {
		if b.Current() < count {
			return r.client.fw.Colorize("red", "%d / %d", b.Current(), count)
		}

		return r.client.fw.Colorize("green", "%d / %d", b.Current(), count)
	})

	uiprogress.Start()
}
