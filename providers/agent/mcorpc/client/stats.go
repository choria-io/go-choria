// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Stats represent stats for a request
type Stats struct {
	RequestID string

	discoveredNodes []string

	outstandingNodes   *NodeList
	unexpectedRespones *NodeList

	responses int32
	passed    int32
	failed    int32

	start time.Time
	end   time.Time

	publishStart time.Time
	publishEnd   time.Time
	publishTotal time.Duration
	publishing   bool

	discoveryStart time.Time
	discoveryEnd   time.Time

	agent  string
	action string

	mu *sync.Mutex
}

// NewStats initializes a new stats instance
func NewStats() *Stats {
	return &Stats{
		discoveredNodes:    []string{},
		outstandingNodes:   NewNodeList(),
		unexpectedRespones: NewNodeList(),
		mu:                 &sync.Mutex{},
	}
}

// Merge merges the stats from a specific batch into this
func (s *Stats) Merge(other *Stats) error {
	if other.All() {
		for _, n := range other.discoveredNodes {
			s.RecordReceived(n)
		}
	} else {
		for _, n := range other.discoveredNodes {
			if !other.outstandingNodes.Have(n) {
				s.RecordReceived(n)
			}
		}
	}

	s.unexpectedRespones.AddHosts(other.UnexpectedResponseFrom()...)

	atomic.AddInt32(&s.passed, other.passed)
	atomic.AddInt32(&s.failed, other.failed)

	d, err := other.PublishDuration()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.publishTotal += d
	s.mu.Unlock()

	return nil
}

// SetAgent stores the agent the stats is for
func (s *Stats) SetAgent(a string) {
	s.agent = a
}

// SetAction stores the action the stats is for
func (s *Stats) SetAction(a string) {
	s.action = a
}

// Agent returns the agent the stat is for if it was set
func (s *Stats) Agent() string {
	return s.agent
}

// Action returns the action the stat is for if it was set
func (s *Stats) Action() string {
	return s.action
}

// All determines if all expected nodes replied already
func (s *Stats) All() bool {
	return s.outstandingNodes.Count() == 0
}

// NoResponseFrom calculates discovered which hosts did not respond
func (s *Stats) NoResponseFrom() []string {
	return s.outstandingNodes.Hosts()
}

// UnexpectedResponseFrom calculates which hosts responses that we did not expect responses from
func (s *Stats) UnexpectedResponseFrom() []string {
	return s.unexpectedRespones.Hosts()
}

// WaitingFor checks if any of the given nodes are still outstanding
func (s *Stats) WaitingFor(nodes []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.outstandingNodes.HaveAny(nodes...)
}

// SetDiscoveredNodes records the node names we expect to communicate with
func (s *Stats) SetDiscoveredNodes(nodes []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.discoveredNodes = nodes

	s.outstandingNodes.Clear()
	s.outstandingNodes.AddHosts(nodes...)
}

// FailedRequestInc increments the failed request counter by one
func (s *Stats) FailedRequestInc() {
	atomic.AddInt32(&s.failed, 1)
}

// PassedRequestInc increments the passed request counter by one
func (s *Stats) PassedRequestInc() {
	atomic.AddInt32(&s.passed, 1)
}

// RecordReceived reords the fact that one message was received
func (s *Stats) RecordReceived(sender string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.AddInt32(&s.responses, 1)

	known := s.outstandingNodes.DeleteIfKnown(sender)
	if !known {
		s.unexpectedRespones.AddHosts(sender)
	}
}

// DiscoveredCount is how many nodes were discovered
func (s *Stats) DiscoveredCount() int {
	return len(s.discoveredNodes)
}

// DiscoveredNodes are the nodes that was discovered for this request
func (s *Stats) DiscoveredNodes() *[]string {
	return &s.discoveredNodes
}

// FailCount is the number of responses that were failures
func (s *Stats) FailCount() int {
	return int(atomic.LoadInt32(&s.failed))
}

// OKCount is the number of responses that were ok
func (s *Stats) OKCount() int {
	return int(atomic.LoadInt32(&s.passed))
}

// ResponsesCount if the total amount of nodes that responded so far
func (s *Stats) ResponsesCount() int {
	return int(atomic.LoadInt32(&s.responses))
}

// StartPublish records the publish process started
func (s *Stats) StartPublish() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.publishing {
		return
	}

	s.publishStart = time.Now()
	s.publishing = true
}

// EndPublish records the publish process ended
func (s *Stats) EndPublish() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.publishing {
		return
	}

	s.publishEnd = time.Now()
	s.publishing = false

	s.publishTotal += s.publishEnd.Sub(s.publishStart)
}

// PublishDuration calculates how long publishing took
func (s *Stats) PublishDuration() (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.publishTotal == 0 || s.publishing {
		return time.Duration(0), fmt.Errorf("publishing is not completed")
	}

	return s.publishTotal, nil
}

// RequestDuration calculates the total duration
func (s *Stats) RequestDuration() (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.start.IsZero() || s.end.IsZero() {
		return time.Duration(0), fmt.Errorf("request is not completed")
	}

	return s.end.Sub(s.start), nil
}

// Start records the start time of a request
func (s *Stats) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.start.IsZero() {
		s.start = time.Now()
	}
}

// Started is the time the request was started, zero time when not started
func (s *Stats) Started() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.start
}

// End records the end time of a request
func (s *Stats) End() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.end.IsZero() {
		s.end = time.Now()
	}
}

// StartDiscover records the start time of the discovery process
func (s *Stats) StartDiscover() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.discoveryStart.IsZero() {
		s.discoveryStart = time.Now()
	}
}

// EndDiscover records the end time of the discovery process
func (s *Stats) EndDiscover() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.discoveryEnd.IsZero() {
		s.discoveryEnd = time.Now()
	}
}

// OverrideDiscoveryTime sets specific discovery time
func (s *Stats) OverrideDiscoveryTime(start time.Time, end time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.discoveryStart = start
	s.discoveryEnd = end
}

// DiscoveryDuration determines how long discovery took, 0 and error when discovery was not done
func (s *Stats) DiscoveryDuration() (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.discoveryStart.IsZero() || s.discoveryEnd.IsZero() {
		return time.Duration(0), fmt.Errorf("discovery was not performed")
	}

	return s.discoveryEnd.Sub(s.discoveryStart), nil
}

// UniqueRequestID is a unique identifier for the request, can be empty
func (s *Stats) UniqueRequestID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.RequestID
}
