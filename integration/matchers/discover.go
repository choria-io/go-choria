// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package matchers

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/gomega"
)

// HaveDiscoverableNodes performs a discovery and asserts that nodes were found
//
//   Expect(client).To(HaveDiscoverableNodes()) - checks for >0 nodes
//   Expect(client).To(HaveDiscoverableNodes(10)) - checks for == 10 nodes
func HaveDiscoverableNodes(flags ...interface{}) gomega.OmegaMatcher {
	m := &haveDiscoveryMatcher{}
	if len(flags) == 1 {
		if v, ok := flags[0].(int); ok {
			m.count = v
		}
	}

	m.ctx, m.cancel = context.WithTimeout(context.Background(), 2*time.Second)

	return m
}

type haveDiscoveryMatcher struct {
	ctx    context.Context
	cancel context.CancelFunc
	count  int
}

type discoverableClient interface {
	DiscoverNodes(ctx context.Context) (nodes []string, err error)
}

func (m *haveDiscoveryMatcher) Match(actual interface{}) (success bool, err error) {
	if m.cancel != nil {
		defer m.cancel()
	}

	client, err := m.toClient(actual)
	if err != nil {
		return false, err
	}

	found, err := client.DiscoverNodes(m.ctx)
	if err != nil {
		return false, fmt.Errorf("discovery failed: %v", err)
	}

	if m.count == 0 {
		return len(found) > 0, nil
	} else {
		return len(found) == m.count, nil
	}
}

func (m *haveDiscoveryMatcher) FailureMessage(_ interface{}) (message string) {
	if m.count == 0 {
		return "Expected to discover 1 or more nodes"
	} else {
		return fmt.Sprintf("Expected to discover %d nodes", m.count)
	}
}

func (m *haveDiscoveryMatcher) NegatedFailureMessage(_ interface{}) (message string) {
	if m.count == 0 {
		return "Expected to not discover any nodes"
	} else {
		return fmt.Sprintf("Expected to not discover %d nodes", m.count)
	}
}

func (m *haveDiscoveryMatcher) toClient(actual interface{}) (discoverableClient, error) {
	response, ok := actual.(discoverableClient)
	if !ok {
		return nil, fmt.Errorf("matcher expects a rpc client that can perform discovery")
	}

	return response, nil
}
