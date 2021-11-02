// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package election

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Backoff controls the interval of campaigns
type Backoff interface {
	// Duration returns the time to sleep for the nth invocation
	Duration(n int) time.Duration
}

// State indicates the current state of the election
type State uint

const (
	// UnknownState indicates the state is unknown, like when the election is not started
	UnknownState State = 0
	// CandidateState is a campaigner that is not the leader
	CandidateState State = 1
	// LeaderState is the leader
	LeaderState State = 2
)

// implements inter.Election
type election struct {
	opts  *options
	state State

	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	lastSeq uint64
	tries   int

	graceCtx    context.Context
	graceCancel context.CancelFunc

	mu            sync.Mutex
	cancelGraceMu sync.Mutex
}

var skipValidate bool

func NewElection(name string, key string, bucket nats.KeyValue, opts ...Option) (*election, error) {
	e := &election{
		state:   UnknownState,
		lastSeq: math.MaxUint64,
		opts: &options{
			name:   name,
			key:    key,
			bucket: bucket,
		},
	}

	status, err := bucket.Status()
	if err != nil {
		return nil, err
	}

	e.opts.ttl = status.TTL()
	if !skipValidate {
		if e.opts.ttl < 30*time.Second {
			return nil, fmt.Errorf("bucket TTL should be 30 seconds or more")
		}
		if e.opts.ttl > time.Hour {
			return nil, fmt.Errorf("bucket TTL should be less than or equal to 1 hour")
		}
	}

	e.opts.cInterval = time.Duration(float64(e.opts.ttl) * 0.75)

	for _, opt := range opts {
		opt(e.opts)
	}

	if !skipValidate {
		if e.opts.cInterval.Seconds() < 5 {
			return nil, fmt.Errorf("campaign interval %v too small", e.opts.cInterval)
		}
		if e.opts.ttl.Seconds()-e.opts.cInterval.Seconds() < 5 {
			return nil, fmt.Errorf("campaign interval %v is too close to bucket ttl %v", e.opts.cInterval, e.opts.ttl)
		}
	}

	e.debugf("Campaign interval: %v", e.opts.cInterval)

	return e, nil
}

func (e *election) debugf(format string, a ...interface{}) {
	if e.opts.debug == nil {
		return
	}
	e.opts.debug(format, a...)
}

func (e *election) campaignForLeadership() error {
	seq, err := e.opts.bucket.Create(e.opts.key, []byte(e.opts.name))
	if err != nil {
		e.tries++
		return nil
	}

	e.lastSeq = seq
	e.state = LeaderState
	e.tries = 0

	// we notify the caller after ~interval so that the past leader has a chance
	// to detect the leadership loss, else if the key got just deleted right
	// before the previous leader did a campaign he will think he is leader
	// for one more round of cInterval
	//
	// cancelGrace interrupts us if a campaign is lost while we were waiting to notify
	// about winning, so we make sure to call lost just in case and only calling win
	// if at the time we're still leader
	go func() {
		e.cancelGraceMu.Lock()
		e.graceCtx, e.graceCancel = context.WithCancel(e.ctx)
		e.cancelGraceMu.Unlock()

		if ctxSleep(e.graceCtx, e.opts.cInterval+50*time.Millisecond) == nil {
			e.mu.Lock()
			defer e.mu.Unlock()

			if e.state == LeaderState {
				if e.opts.wonCb != nil {
					e.opts.wonCb()
				}
			} else {
				if e.opts.lostCb != nil {
					e.opts.lostCb()
				}
			}
		}
	}()

	return nil
}

func (e *election) maintainLeadership() error {
	seq, err := e.opts.bucket.Update(e.opts.key, []byte(e.opts.name), e.lastSeq)
	if err != nil {
		e.debugf("key update failed, moving to candidate state: %v", err)
		e.state = CandidateState
		e.lastSeq = math.MaxUint64

		// stop our grace period notifications
		e.cancelGraceMu.Lock()
		if e.graceCancel != nil {
			e.graceCancel()
		}
		e.cancelGraceMu.Unlock()

		if e.opts.lostCb != nil {
			e.opts.lostCb()
		}

		return err
	}
	e.lastSeq = seq

	return nil
}

func (e *election) try() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.opts.campaignCb != nil {
		e.opts.campaignCb(e.state)
	}

	switch e.state {
	case LeaderState:
		return e.maintainLeadership()

	case CandidateState:
		return e.campaignForLeadership()

	default:
		return fmt.Errorf("campaigned while in unknown state")
	}
}

func (e *election) campaign(wg *sync.WaitGroup) error {
	defer wg.Done()

	e.mu.Lock()
	e.state = CandidateState
	e.mu.Unlock()

	// spread out startups a bit
	splay := time.Duration(rand.Intn(int(e.opts.cInterval.Milliseconds())))
	ctxSleep(e.ctx, splay)

	var ticker *time.Ticker
	if e.opts.bo != nil {
		ticker = time.NewTicker(e.opts.bo.Duration(0))
	} else {
		ticker = time.NewTicker(e.opts.cInterval)
	}

	tick := func() {
		err := e.try()
		if err != nil {
			e.debugf("election attempt failed: %v", err)
		}

		if e.opts.bo != nil {
			ticker.Reset(e.opts.bo.Duration(e.tries))
		}
	}

	// initial campaign
	tick()

	for {
		select {
		case <-ticker.C:
			tick()

		case <-e.ctx.Done():
			ticker.Stop()
			e.stop()

			if e.opts.lostCb != nil && e.IsLeader() {
				e.debugf("Calling leader lost during shutdown")
				e.opts.lostCb()
			}

			return nil
		}
	}
}

func (e *election) stop() {
	e.mu.Lock()
	e.started = false
	e.cancel()
	e.state = CandidateState
	e.lastSeq = math.MaxUint64
	e.mu.Unlock()
}

func (e *election) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return fmt.Errorf("already running")
	}

	e.ctx, e.cancel = context.WithCancel(ctx)
	e.started = true
	e.mu.Unlock()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	err := e.campaign(wg)
	if err != nil {
		e.stop()
		return err
	}

	wg.Wait()

	return nil
}

func (e *election) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.started {
		return
	}

	if e.cancel != nil {
		e.cancel()
	}

	e.stop()
}

func (e *election) IsLeader() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.state == LeaderState
}

func (e *election) State() State {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.state
}

func ctxSleep(ctx context.Context, duration time.Duration) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	sctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	<-sctx.Done()

	return ctx.Err()
}
