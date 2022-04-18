// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package matchers

import (
	"fmt"

	"github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/onsi/gomega/types"
)

// HaveUnexpectedResponders asserts that there were count unexpected responders
func HaveUnexpectedResponders(count int) types.GomegaMatcher {
	return &haveResponsesMatcher{responses: count, unexpected: true}
}

// HaveAnyUnexpectedResponders asserts that any number >0 unexpected responses were received
func HaveAnyUnexpectedResponders() types.GomegaMatcher {
	return &haveResponsesMatcher{responses: -1, unexpected: true}
}

// HaveNoUnexpectedResponders asserts that no unexpected responses were received
func HaveNoUnexpectedResponders() types.GomegaMatcher {
	return &haveResponsesMatcher{responses: 0, unexpected: true}
}

// HaveAllResponses asserts that all discovered nodes responded
func HaveAllResponses() types.GomegaMatcher {
	return &haveResponsesMatcher{all: true}
}

// HaveSuccessfulResponses asserts that count successful responses were received
func HaveSuccessfulResponses(count int) types.GomegaMatcher {
	return &haveResponsesMatcher{responses: count, success: true}
}

// HaveOnlySuccessfulResponses asserts that more than 1 response were received and that all were successful
func HaveOnlySuccessfulResponses() types.GomegaMatcher {
	return &haveResponsesMatcher{responses: -1, success: true}
}

// HaveFailedResponses asserts that count failed responses were received
func HaveFailedResponses(count int) types.GomegaMatcher {
	return &haveResponsesMatcher{responses: count, failed: true}
}

// HaveOnlyFailedResponses asserts that more than 1 responses were received and that all responses were failures
func HaveOnlyFailedResponses() types.GomegaMatcher {
	return &haveResponsesMatcher{responses: -1, failed: true}
}

// HaveResponses asserts that count responses of any response code was received
func HaveResponses(count int) types.GomegaMatcher {
	return &haveResponsesMatcher{responses: count}
}

type statsProvider interface {
	RPCClientStats() *client.Stats
}

type haveResponsesMatcher struct {
	responses  int
	failed     bool
	success    bool
	all        bool
	unexpected bool
}

func (m *haveResponsesMatcher) Match(actual interface{}) (success bool, err error) {
	stats, err := m.toStats(actual)
	if err != nil {
		return false, err
	}

	switch {
	case m.unexpected:
		if m.responses < 0 {
			return len(stats.UnexpectedResponseFrom()) > 0, nil
		} else {
			return len(stats.UnexpectedResponseFrom()) == m.responses, nil
		}

	case m.all:
		return stats.All(), nil
	case m.failed:
		if m.responses < 0 {
			return stats.ResponsesCount() > 0 && stats.FailCount() == m.responses, nil
		} else {
			return stats.FailCount() == m.responses, nil
		}

	case m.success:
		if m.responses < 0 {
			return stats.ResponsesCount() > 0 && stats.OKCount() == stats.ResponsesCount(), nil
		} else {
			return stats.OKCount() == m.responses, nil
		}

	default:
		return stats.ResponsesCount() == m.responses, nil
	}
}

func (m *haveResponsesMatcher) FailureMessage(actual interface{}) (message string) {
	stats, _ := m.toStats(actual)

	responses := "responses"
	if m.responses == 1 {
		responses = "response"
	}

	switch {
	case m.unexpected:
		if m.responses < 0 {
			return fmt.Sprintf("Expected unexpected responses")
		} else {
			return fmt.Sprintf("Expected %d unexpected responses, had %d", m.responses, len(stats.UnexpectedResponseFrom()))
		}

	case m.all:
		return fmt.Sprintf("Expected all discovered nodes to have responded, %d / %d received", stats.ResponsesCount(), stats.DiscoveredCount())

	case m.failed:
		expected := m.responses
		if m.responses < 0 {
			expected = stats.ResponsesCount()
		}

		return fmt.Sprintf("Expected to have %d failed %s, got %d", expected, responses, stats.FailCount())

	case m.success:
		expected := m.responses
		if m.responses < 0 {
			expected = stats.ResponsesCount()
		}

		return fmt.Sprintf("Expected to have %d OK %s, got %d", expected, responses, stats.OKCount())

	default:
		return fmt.Sprintf("Expected  to have %d %s, got %d", m.responses, responses, stats.ResponsesCount())
	}
}

func (m *haveResponsesMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	stats, _ := m.toStats(actual)

	responses := "responses"
	if m.responses == 1 {
		responses = "response"
	}

	switch {
	case m.unexpected:
		if m.responses < 0 {
			return fmt.Sprintf("Expected no unexpected responders")
		} else {
			return fmt.Sprintf("Expected all discovered nodes to not have responded, %d / %d received", stats.ResponsesCount(), stats.DiscoveredCount())
		}

	case m.all:
		return fmt.Sprintf("Expected all discovered nodes to not have responded, %d / %d received", stats.ResponsesCount(), stats.DiscoveredCount())
	case m.failed:
		return fmt.Sprintf("Expected to not have failed %d %s", m.responses, responses)
	case m.success:
		if m.responses < 0 {
			return fmt.Sprintf("Expected to not have %d OK %s", stats.ResponsesCount(), responses)
		} else {
			return fmt.Sprintf("Expected to not have %d OK %s", m.responses, responses)
		}

	default:
		return fmt.Sprintf("Expected to not have %d %s", m.responses, responses)
	}
}

func (m *haveResponsesMatcher) toStats(actual interface{}) (*client.Stats, error) {
	response, ok := actual.(statsProvider)
	if !ok {
		return nil, fmt.Errorf("matcher expects a rpc response that can provide statistics")
	}

	return response.RPCClientStats(), nil
}
