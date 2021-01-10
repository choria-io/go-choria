// generated code; DO NOT EDIT

package choria_utilclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/client/discovery/broadcast"
	"github.com/choria-io/go-choria/client/discovery/puppetdb"
	"github.com/choria-io/go-choria/protocol"
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
func (b *BroadcastNS) Discover(ctx context.Context, fw ChoriaFramework, filters []FilterFunc) ([]string, error) {
	b.Lock()
	defer b.Unlock()

	copier := func() []string {
		out := make([]string, len(b.nodeCache))
		for i, n := range b.nodeCache {
			out[i] = n
		}

		return out
	}

	if !(b.nodeCache == nil || len(b.nodeCache) == 0) {
		return copier(), nil
	}

	err := b.parseFilters(filters)
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

func (p *BroadcastNS) parseFilters(fs []FilterFunc) error {
	p.f = protocol.NewFilter()

	for _, f := range fs {
		err := f(p.f)
		if err != nil {
			return err
		}
	}

	return nil
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
func (p *PuppetDBNS) Discover(ctx context.Context, fw ChoriaFramework, filters []FilterFunc) ([]string, error) {
	p.Lock()
	defer p.Unlock()

	copier := func() []string {
		out := make([]string, len(p.nodeCache))
		for i, n := range p.nodeCache {
			out[i] = n
		}

		return out
	}

	if !(p.nodeCache == nil || len(p.nodeCache) == 0) {
		return copier(), nil
	}

	err := p.parseFilters(filters)
	if err != nil {
		return nil, err
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

func (p *PuppetDBNS) parseFilters(fs []FilterFunc) error {
	p.f = protocol.NewFilter()

	for _, f := range fs {
		err := f(p.f)
		if err != nil {
			return err
		}
	}

	if len(p.f.Compound) > 0 {
		return fmt.Errorf("compound filters are not supported by PuppetDB")
	}

	return nil
}
