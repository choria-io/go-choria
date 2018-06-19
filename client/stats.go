package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/atomic"
)

// Stats represent stats for a request
type Stats struct {
	RequestID string

	discoveredNodes []string

	outstandingNodes   *NodeList
	unexpectedRespones *NodeList

	responses atomic.Int32
	passed    atomic.Int32
	failed    atomic.Int32

	start time.Time
	end   time.Time

	publishStart time.Time
	publishEnd   time.Time
	publishTotal time.Duration
	publishing   bool

	discoveryStart time.Time
	discoveryEnd   time.Time

	mu   *sync.Mutex
	once sync.Once
}

// NewStats initializes a new stats instance
func NewStats() *Stats {
	return &Stats{
		discoveredNodes:    []string{},
		outstandingNodes:   NewNodeList(),
		unexpectedRespones: NewNodeList(),
		mu:                 &sync.Mutex{},
		publishTotal:       0,
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

	s.passed.Add(other.passed.Load())
	s.failed.Add(other.failed.Load())

	d, err := other.PublishDuration()
	if err != nil {
		return err
	}

	s.publishTotal += d

	return nil
}

func (s *Stats) showProgress(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			discovered := s.DiscoveredCount()

			fmt.Printf("ok: %-5d failed: %-5d received: %d / %d\n", s.passed.Load(), s.failed.Load(), s.ResponsesCount(), discovered)
		case <-ctx.Done():
			return
		}
	}
}

// All determines if all expected nodes replied already
func (s *Stats) All() bool {
	return s.outstandingNodes.Count() == 0
}

// StartProgress starts a basic progress display that will be interrupted by the context
func (s *Stats) StartProgress(ctx context.Context) {
	s.once.Do(func() { go s.showProgress(ctx) })
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

	if s.outstandingNodes.HaveAny(nodes...) {
		return true
	}

	return false
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
	s.failed.Inc()
}

// PassedRequestInc increments the passed request counter by one
func (s *Stats) PassedRequestInc() {
	s.passed.Inc()
}

// RecordReceived reords the fact that one message was received
func (s *Stats) RecordReceived(sender string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.responses.Inc()

	known := s.outstandingNodes.DeleteIfKnown(sender)
	if !known {
		s.unexpectedRespones.AddHosts(sender)
	}
}

// DiscoveredCount is how many nodes were discovered
func (s *Stats) DiscoveredCount() int {
	return len(s.discoveredNodes)
}

// FailCount is the number of responses that were failures
func (s *Stats) FailCount() int {
	return int(s.failed.Load())
}

// OKCount is the number of responses that were ok
func (s *Stats) OKCount() int {
	return int(s.passed.Load())
}

// ResponsesCount if the total amount of nodes that responded so far
func (s *Stats) ResponsesCount() int {
	return int(s.responses.Load())
}

// StartPublish records the publish started
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

// DiscoveryDuration determines how long discovery took, 0 and error when discovery was not done
func (s *Stats) DiscoveryDuration() (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.discoveryStart.IsZero() || s.discoveryEnd.IsZero() {
		return time.Duration(0), fmt.Errorf("discovery was not performed")
	}

	return s.discoveryEnd.Sub(s.discoveryStart), nil
}
