// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// generated code; DO NOT EDIT

package aaa_signerclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/discovery/broadcast"
	"github.com/choria-io/go-choria/providers/discovery/external"
	"github.com/choria-io/go-choria/providers/discovery/puppetdb"
)

// BroadcastNS is a NodeSource that uses the Choria network broadcast method to discover nodes
type BroadcastNS struct {
	nodeCache []string
	f         *protocol.Filter

	sync.Mutex
}

// Reset resets the internal node cache
func (b *BroadcastNS) Reset() {
	b.Lock()
	defer b.Unlock()

	b.nodeCache = []string{}
}

// Discover performs the discovery of nodes against the Choria Network
func (b *BroadcastNS) Discover(ctx context.Context, fw inter.Framework, filters []FilterFunc) ([]string, error) {
	b.Lock()
	defer b.Unlock()

	copier := func() []string {
		out := make([]string, len(b.nodeCache))
		copy(out, b.nodeCache)

		return out
	}

	if !(b.nodeCache == nil || len(b.nodeCache) == 0) {
		return copier(), nil
	}

	var err error

	b.f, err = parseFilters(filters)
	if err != nil {
		return nil, err
	}

	if b.nodeCache == nil {
		b.nodeCache = []string{}
	}

	cfg := fw.Configuration()
	nodes, err := broadcast.New(fw).Discover(ctx, broadcast.Filter(b.f), broadcast.Timeout(time.Second*time.Duration(cfg.DiscoveryTimeout)))
	if err != nil {
		return []string{}, err
	}

	b.nodeCache = nodes

	return copier(), nil
}

// ExternalNS is a NodeSource that calls an external command for discovery
type ExternalNS struct {
	nodeCache []string
	f         *protocol.Filter

	sync.Mutex
}

// Reset resets the internal node cache
func (p *ExternalNS) Reset() {
	p.Lock()
	defer p.Unlock()

	p.nodeCache = []string{}
}

func (p *ExternalNS) Discover(ctx context.Context, fw inter.Framework, filters []FilterFunc) ([]string, error) {
	p.Lock()
	defer p.Unlock()

	copier := func() []string {
		out := make([]string, len(p.nodeCache))
		copy(out, p.nodeCache)

		return out
	}

	if !(p.nodeCache == nil || len(p.nodeCache) == 0) {
		return copier(), nil
	}

	var err error
	p.f, err = parseFilters(filters)
	if err != nil {
		return nil, err
	}

	if p.nodeCache == nil {
		p.nodeCache = []string{}
	}

	cfg := fw.Configuration()
	nodes, err := external.New(fw).Discover(ctx, external.Filter(p.f), external.Timeout(time.Second*time.Duration(cfg.DiscoveryTimeout)))
	if err != nil {
		return []string{}, err
	}

	p.nodeCache = nodes

	return copier(), nil
}

// PuppetDBNS is a NodeSource that uses the PuppetDB PQL Queries to discover nodes
type PuppetDBNS struct {
	nodeCache []string
	f         *protocol.Filter

	sync.Mutex
}

// Reset resets the internal node cache
func (p *PuppetDBNS) Reset() {
	p.Lock()
	defer p.Unlock()

	p.nodeCache = []string{}
}

// Discover performs the discovery of nodes against the Choria Network
func (p *PuppetDBNS) Discover(ctx context.Context, fw inter.Framework, filters []FilterFunc) ([]string, error) {
	p.Lock()
	defer p.Unlock()

	copier := func() []string {
		out := make([]string, len(p.nodeCache))
		copy(out, p.nodeCache)

		return out
	}

	if !(p.nodeCache == nil || len(p.nodeCache) == 0) {
		return copier(), nil
	}

	var err error
	p.f, err = parseFilters(filters)
	if err != nil {
		return nil, err
	}

	if len(p.f.Compound) > 0 {
		return nil, fmt.Errorf("compound filters are not supported by PuppetDB")
	}

	if p.nodeCache == nil {
		p.nodeCache = []string{}
	}

	cfg := fw.Configuration()
	nodes, err := puppetdb.New(fw).Discover(ctx, puppetdb.Filter(p.f), puppetdb.Timeout(time.Second*time.Duration(cfg.DiscoveryTimeout)))
	if err != nil {
		return []string{}, err
	}

	p.nodeCache = nodes

	return copier(), nil
}

// MetaNS is a NodeSource that assists CLI tools in creating Choria standard command line based discovery.
type MetaNS struct {
	// Options is the CLI options to discover based on
	Options *discovery.StandardOptions

	// Agent should be the agent the request is targeted at
	Agent string

	// DisablePipedDiscovery prevents the STDIN being used as a discovery source
	DisablePipedDiscovery bool

	nodeCache []string
	sync.Mutex
}

// NewMetaNS creates a new meta discovery node source
func NewMetaNS(opts *discovery.StandardOptions, enablePipeMode bool) *MetaNS {
	return &MetaNS{
		Options:               opts,
		Agent:                 "aaa_signer",
		DisablePipedDiscovery: !enablePipeMode,
		nodeCache:             []string{},
	}
}

// Reset resets the internal node cache
func (p *MetaNS) Reset() {
	p.Lock()
	defer p.Unlock()

	p.nodeCache = []string{}
}

// Discover performs the discovery of nodes against the Choria Network.
func (p *MetaNS) Discover(ctx context.Context, fw inter.Framework, _ []FilterFunc) ([]string, error) {
	p.Lock()
	defer p.Unlock()

	copier := func() []string {
		out := make([]string, len(p.nodeCache))
		copy(out, p.nodeCache)

		return out
	}

	if !(p.nodeCache == nil || len(p.nodeCache) == 0) {
		return copier(), nil
	}

	if p.nodeCache == nil {
		p.nodeCache = []string{}
	}

	if p.Options == nil {
		return nil, fmt.Errorf("options have not been set")
	}

	nodes, _, err := p.Options.Discover(ctx, fw, p.Agent, !p.DisablePipedDiscovery, false, fw.Logger("discovery"))
	if err != nil {
		return nil, err
	}

	p.nodeCache = nodes

	return copier(), nil
}

func parseFilters(fs []FilterFunc) (*protocol.Filter, error) {
	filter := protocol.NewFilter()

	for _, f := range fs {
		err := f(filter)
		if err != nil {
			return nil, err
		}
	}

	return filter, nil
}
