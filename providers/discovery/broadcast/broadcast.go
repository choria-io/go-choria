// Package broadcast implements a MCollective like broadcast discovery system for nodes running choria
//
// It is not thread safe and a single instance of the discoverer shouldn't be shared by go routines etc, you can reuse them but should
// not be using the same one multiple times.
//
// It will create a single connection to your Choria network and close it once the context to Discover is canceled.
//
// It has been shown to discover 50 000 nodes in around 1.2 seconds, I'd suggest on such a large network setting
// protocol.ClientStrictValidation to false
package broadcast

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/sirupsen/logrus"
)

// Broadcast implements mcollective like broadcast discovery
type Broadcast struct {
	fw      client.ChoriaFramework
	timeout time.Duration
	log     *logrus.Entry
}

// ChoriaClient implements the connection to the Choria network
type ChoriaClient interface {
	Request(ctx context.Context, msg inter.Message, handler client.Handler) (err error)
}

var (
	stringIsRegex = regexp.MustCompile(`^/(.+)/$`)
)

// New creates a new broadcast discovery client
func New(fw client.ChoriaFramework) *Broadcast {
	b := &Broadcast{
		fw:      fw,
		timeout: time.Second * time.Duration(fw.Configuration().DiscoveryTimeout),
		log:     fw.Logger("broadcast_discovery"),
	}

	return b
}

// Discover performs a broadcast discovery using the supplied filter
func (b *Broadcast) Discover(ctx context.Context, opts ...DiscoverOption) (n []string, err error) {
	dopts := &dOpts{
		collective:     b.fw.Configuration().MainCollective,
		discovered:     []string{},
		filter:         protocol.NewFilter(),
		mu:             &sync.Mutex{},
		timeout:        b.timeout,
		dynamicTimeout: b.fw.Configuration().Choria.BroadcastDiscoveryDynamicTimeout,
	}

	for _, opt := range opts {
		opt(dopts)
	}

	if dopts.cl == nil {
		opts := []client.Option{
			client.Receivers(3),
			client.Timeout(dopts.timeout),
		}

		if dopts.name != "" {
			opts = append(opts, client.Name(dopts.name))
		}

		dopts.cl, err = client.New(b.fw, opts...)
		if err != nil {
			return n, fmt.Errorf("could not create choria client: %s", err)
		}
	}

	if b.identityOptimize(dopts.filter) {
		b.log.Debugf("Performing identity-only filter optimization")
		return dopts.filter.IdentityFilters(), nil
	}

	dopts.msg, err = b.createMessage(dopts.filter, dopts.collective)
	if err != nil {
		return n, fmt.Errorf("could not create message: %s", err)
	}

	b.log.Debugf("Performing broadcast discovery")

	// wrapping it ensures the initial connection does not run forever and inherits the parent ^C handling etc
	// the +2 gives some additional time to the whole request for network connect time etc
	rctx, cancel := context.WithTimeout(ctx, dopts.timeout+2)
	defer cancel()

	err = dopts.cl.Request(rctx, dopts.msg, b.handler(ctx, cancel, dopts))
	if err != nil {
		return n, fmt.Errorf("could not perform request: %s", err)
	}

	return dopts.discovered, nil
}

func (b *Broadcast) identityOptimize(filter *protocol.Filter) bool {
	if !(len(filter.CompoundFilters()) == 0 && len(filter.FactFilters()) == 0 && len(filter.ClassFilters()) == 0 && len(filter.IdentityFilters()) > 0) {
		return false
	}

	for _, f := range filter.IdentityFilters() {
		if stringIsRegex.MatchString(f) {
			return false
		}
	}

	return true
}

func (b *Broadcast) createMessage(filter *protocol.Filter, collective string) (inter.Message, error) {
	msg, err := b.fw.NewMessage(base64.StdEncoding.EncodeToString([]byte("ping")), "discovery", collective, "request", nil)
	if err != nil {
		return nil, fmt.Errorf("could not create message: %s", err)
	}

	msg.SetProtocolVersion(protocol.RequestV1)
	msg.SetReplyTo(msg.ReplyTarget())
	msg.SetFilter(filter)

	return msg, err
}

func (b *Broadcast) handler(ctx context.Context, cancel func(), dopts *dOpts) client.Handler {
	var timer *time.Timer
	dynamicTimeout := 300 * time.Millisecond
	if dopts.dynamicTimeout {
		b.log.Debugf("Configuring dynamic discovery timeout")
		timer = time.NewTimer(b.timeout)
		go func() {
			select {
			case <-timer.C:
				cancel()
			case <-ctx.Done():
			}

			timer.Stop()
		}()
	}

	return func(_ context.Context, m inter.ConnectorMessage) {
		reply, err := b.fw.NewTransportFromJSON(string(m.Data()))
		if err != nil {
			b.log.Errorf("Could not process a reply: %s", err)
			return
		}

		dopts.mu.Lock()
		defer dopts.mu.Unlock()

		dopts.discovered = append(dopts.discovered, reply.SenderID())
		if dopts.dynamicTimeout {
			timer.Reset(dynamicTimeout)
		}
	}
}
