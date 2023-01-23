// Copyright 2020-2022 The NATS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package governor controls the concurrency of a network wide process
//
// Using this one can, for example, create CRON jobs that can trigger
// 100s or 1000s concurrently but where most will wait for a set limit
// to complete.  In effect limiting the overall concurrency of these
// execution.
//
// To do this a Stream is created that has a maximum message limit and
// that will reject new entries when full.
//
// Workers will try to place themselves in the Stream, they do their work
// if they succeed and remove themselves from the Stream once they are done.
//
// As a fail safe the stack will evict entries after a set time based on
// Stream max age.
//
// A manager is included to create, observe and edit these streams and the
// choria CLI has a new command build on this library: choria governor
package governor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/backoff"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/google/go-cmp/cmp"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
)

// DefaultInterval default sleep between tries, set with WithInterval()
const DefaultInterval = 250 * time.Millisecond

// Finisher signals that work is completed releasing the slot on the stack
type Finisher func() error

// Governor controls concurrency of distributed processes using a named governor stream
type Governor interface {
	// Start attempts to get a spot in the Governor, gives up on context, call Finisher to signal end of work
	Start(ctx context.Context, name string) (fin Finisher, seq uint64, err error)
	// Connection is the NATS connection used to communicate
	Connection() *nats.Conn
}

// Logger is a custom logger
type Logger interface {
	Debugf(format string, a ...any)
	Infof(format string, a ...any)
	Warnf(format string, a ...any)
	Errorf(format string, a ...any)
}

// Manager controls concurrent executions of work distributed throughout a nats network by using
// a stream as a capped stack where workers reserve a slot and later release the slot
type Manager interface {
	// Limit is the configured maximum entries in the Governor
	Limit() int64
	// MaxAge is the time after which entries will be evicted
	MaxAge() time.Duration
	// Name is the Governor name
	Name() string
	// Replicas is how many data replicas are kept of the data
	Replicas() int
	// SetLimit configures the maximum entries in the Governor and takes immediate effect
	SetLimit(uint64) error
	// SetMaxAge configures the maximum age of entries, takes immediate effect
	SetMaxAge(time.Duration) error
	// SetSubject configures the underlying NATS subject the Governor listens on for entry campaigns
	SetSubject(subj string) error
	// Stream is the underlying JetStream stream
	Stream() *jsm.Stream
	// Subject is the subject the Governor listens on for entry campaigns
	Subject() string
	// Reset resets the governor removing all current entries from it
	Reset() error
	// Active is the number of active entries in the Governor
	Active() (uint64, error)
	// Evict removes an entry from the Governor given its unique id, returns the name that was on that entry
	Evict(entry uint64) (name string, err error)
	// LastActive returns the the since entry was added to the Governor, can be zero time when no entries were added
	LastActive() (time.Time, error)
	// Connection is the NATS connection used to communicate
	Connection() *nats.Conn
}

var errRetry = errors.New("retryable error")

type jsGMgr struct {
	name     string
	stream   string
	maxAge   time.Duration
	limit    uint64
	mgr      *jsm.Manager
	nc       *nats.Conn
	str      *jsm.Stream
	subj     string
	replicas int
	running  bool
	noCreate bool
	noLeave  bool

	logger Logger
	cint   time.Duration
	bo     *backoff.Policy

	mu sync.Mutex
}

func NewManager(name string, limit uint64, maxAge time.Duration, replicas uint, nc *nats.Conn, update bool, opts ...Option) (Manager, error) {
	mgr, err := jsm.New(nc)
	if err != nil {
		return nil, err
	}

	gov := &jsGMgr{
		name:     name,
		maxAge:   maxAge,
		limit:    limit,
		mgr:      mgr,
		nc:       nc,
		replicas: int(replicas),
		cint:     DefaultInterval,
	}

	for _, opt := range opts {
		opt(gov)
	}

	if limit == 0 {
		gov.noCreate = true
	}

	gov.stream = gov.streamName()
	gov.subj = gov.streamSubject()

	err = gov.loadOrCreate(update)
	if err != nil {
		return nil, err
	}

	return gov, nil
}

type Option func(mgr *jsGMgr)

// WithLogger configures the logger to use, no logging when none is given
func WithLogger(log Logger) Option {
	return func(mgr *jsGMgr) {
		mgr.logger = log
	}
}

// WithBackoff sets a backoff policy for gradually reducing try interval
func WithBackoff(p backoff.Policy) Option {
	return func(mgr *jsGMgr) {
		mgr.bo = &p
	}
}

// WithInterval sets the interval between tries
func WithInterval(i time.Duration) Option {
	return func(mgr *jsGMgr) {
		mgr.cint = i
	}
}

// WithSubject configures a specific subject for the governor to act on
func WithSubject(s string) Option {
	return func(mgr *jsGMgr) {
		mgr.subj = s
	}
}

// WithoutLeavingOnCompletion prevents removal from the governor after execution
func WithoutLeavingOnCompletion() Option {
	return func(mgr *jsGMgr) {
		mgr.noLeave = true
	}
}

func New(name string, nc *nats.Conn, opts ...Option) Governor {
	mgr, err := jsm.New(nc)
	if err != nil {
		return nil
	}

	gov := &jsGMgr{
		name: name,
		mgr:  mgr,
		nc:   nc,
		cint: DefaultInterval,
	}

	for _, opt := range opts {
		opt(gov)
	}

	gov.stream = gov.streamName()
	gov.subj = gov.streamSubject()

	return gov
}

func (g *jsGMgr) streamSubject() string {
	if g.subj != "" {
		return g.subj
	}

	return fmt.Sprintf("$GOVERNOR.campaign.%s", g.name)
}

func (g *jsGMgr) streamName() string {
	if g.stream != "" {
		return g.stream
	}

	return StreamName(g.name)
}

func StreamName(governor string) string {
	return fmt.Sprintf("GOVERNOR_%s", governor)
}

func List(nc *nats.Conn, collective string) ([]string, error) {
	mgr, err := jsm.New(nc)
	if err != nil {
		return nil, err
	}

	known, err := mgr.StreamNames(&jsm.StreamNamesFilter{
		Subject: iu.GovernorSubject("*", collective),
	})
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(known); i++ {
		known[i] = strings.TrimPrefix(known[i], "GOVERNOR_")
	}

	sort.Strings(known)

	return known, nil
}
func (g *jsGMgr) Start(ctx context.Context, name string) (Finisher, uint64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.running {
		return nil, 0, fmt.Errorf("already running")
	}

	g.running = true
	seq := uint64(0)
	tries := 0

	try := func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		g.Debugf("Publishing to %s", g.subj)
		m, err := g.nc.RequestWithContext(ctx, g.subj, []byte(name))
		if err != nil {
			g.Errorf("Publishing to governor %s via %s failed: %s", g.name, g.subj, err)
			return err
		}

		res, err := jsm.ParsePubAck(m)
		if err != nil {
			// jetstream sent us a puback error, this is retryable in the case of governors
			if jsm.IsNatsError(err, 10077) {
				g.Debugf("Could not obtain a slot: %v", err)
				return errRetry
			}

			g.Errorf("Invalid pub ack: %s", err)
			return err
		}

		seq = res.Sequence

		g.Infof("Got a slot on %s with sequence %d", g.name, seq)

		return nil
	}

	closer := func() error {
		if seq == 0 {
			return nil
		}

		g.mu.Lock()
		defer g.mu.Unlock()
		if !g.running {
			return nil
		}

		g.running = false

		if g.noLeave {
			g.Infof("Not evicting self from %s based on configuration directive", g.name)
			return nil
		}

		g.Infof("Removing self from %s sequence %d", g.name, seq)
		err := g.mgr.DeleteStreamMessage(g.stream, seq, true)
		if err != nil {
			g.Errorf("Could not remove self from %s: %s", g.name, err)
			return fmt.Errorf("could not remove seq %d: %s", seq, err)
		}

		return nil
	}

	g.Debugf("Starting to campaign every %v for a slot on %s using %s", g.cint, g.name, g.subj)

	// we try to enter the governor and if it fails in a way thats safe to retry
	// we will do so else we exit.
	//
	// We need to handle thins like context timeout, bucket not found etc specifically
	// as hard errors since, especially context timeout, it does not mean the message did
	// not enter the governor, it just means something went wrong, perhaps in getting the
	// ok reply.  In the case where the message did reach the governor but the reply could
	// not be processed we will retry again and again potentially filling the governor.
	err := try()
	if err == nil {
		return closer, seq, nil
	} else if err != errRetry {
		return nil, 0, err
	}

	ticker := time.NewTicker(g.cint)

	for {
		select {
		case <-ticker.C:
			tries++

			err = try()
			if err == nil {
				return closer, seq, nil
			} else if err != errRetry {
				return nil, 0, err
			}

			if g.bo != nil {
				delay := g.bo.Duration(tries)
				g.Debugf("Retrying after %v", delay)
				ticker.Reset(delay)
			}

		case <-ctx.Done():
			g.Infof("Stopping campaigns against %s due to context timeout after %d tries", g.name, tries)
			ticker.Stop()
			return nil, 0, ctx.Err()
		}
	}
}

func (g *jsGMgr) Reset() error {
	return g.str.Purge()
}
func (g *jsGMgr) Stream() *jsm.Stream    { return g.str }
func (g *jsGMgr) Limit() int64           { return g.str.MaxMsgs() }
func (g *jsGMgr) MaxAge() time.Duration  { return g.str.MaxAge() }
func (g *jsGMgr) Subject() string        { return g.str.Subjects()[0] }
func (g *jsGMgr) Replicas() int          { return g.str.Replicas() }
func (g *jsGMgr) Connection() *nats.Conn { return g.nc }
func (g *jsGMgr) Name() string           { return g.name }
func (g *jsGMgr) Evict(entry uint64) (string, error) {
	msg, err := g.str.ReadMessage(entry)
	if err != nil {
		return "", err
	}

	return string(msg.Data), g.str.DeleteMessage(entry)
}

func (g *jsGMgr) Active() (uint64, error) {
	nfo, err := g.str.Information()
	if err != nil {
		return 0, err
	}

	return nfo.State.Msgs, nil
}

func (g *jsGMgr) LastActive() (time.Time, error) {
	nfo, err := g.str.Information()
	if err != nil {
		return time.Time{}, err
	}

	return nfo.State.LastTime, nil
}

func (g *jsGMgr) SetSubject(subj string) error {
	g.mu.Lock()
	g.subj = subj
	g.mu.Unlock()

	return g.updateConfig()
}

func (g *jsGMgr) SetLimit(limit uint64) error {
	g.mu.Lock()
	g.limit = limit
	g.mu.Unlock()

	return g.updateConfig()
}

func (g *jsGMgr) SetMaxAge(age time.Duration) error {
	g.mu.Lock()
	g.maxAge = age
	g.mu.Unlock()

	return g.updateConfig()
}

func (g *jsGMgr) updateConfig() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.str.MaxAge() != g.maxAge || g.str.MaxMsgs() != int64(g.limit) || !cmp.Equal([]string{g.streamSubject()}, g.str.Subjects()) || g.str.Replicas() != g.replicas {
		err := g.str.UpdateConfiguration(g.str.Configuration(), g.streamOpts()...)
		if err != nil {
			return fmt.Errorf("stream update failed: %s", err)
		}
	}

	return nil
}

func (g *jsGMgr) streamOpts() []jsm.StreamOption {
	opts := []jsm.StreamOption{
		jsm.StreamDescription(fmt.Sprintf("Concurrency Governor %s", g.name)),
		jsm.MaxAge(g.maxAge),
		jsm.MaxMessages(int64(g.limit)),
		jsm.Subjects(g.subj),
		jsm.Replicas(g.replicas),
		jsm.LimitsRetention(),
		jsm.FileStorage(),
		jsm.DiscardNew(),
		jsm.DuplicateWindow(0),
	}

	if g.replicas > 0 {
		opts = append(opts, jsm.Replicas(g.replicas))
	}

	return opts
}

func (g *jsGMgr) loadOrCreate(update bool) error {
	opts := g.streamOpts()

	if g.noCreate {
		has, err := g.mgr.IsKnownStream(g.stream)
		if err != nil {
			return err
		}

		if !has {
			return fmt.Errorf("unknown governor")
		}
	}

	str, err := g.mgr.LoadOrNewStream(g.stream, opts...)
	if err != nil {
		return err
	}

	g.str = str

	if update {
		g.updateConfig()
	}

	return nil
}

func (g *jsGMgr) Debugf(format string, a ...any) {
	if g.logger != nil {
		g.logger.Debugf(format, a...)
	}
}

func (g *jsGMgr) Infof(format string, a ...any) {
	if g.logger != nil {
		g.logger.Infof(format, a...)
	}
}

func (g *jsGMgr) Warnf(format string, a ...any) {
	if g.logger != nil {
		g.logger.Warnf(format, a...)
	}
}

func (g *jsGMgr) Errorf(format string, a ...any) {
	if g.logger != nil {
		g.logger.Errorf(format, a...)
	}
}
