package agents

import (
	"regexp"
	"strings"
)

// Match agents on a AND basis
func Match(needles []string, knownAgents []string) bool {
	matched := 0
	failed := 0

	for _, needle := range needles {
		if strings.HasPrefix(needle, "/") && strings.HasSuffix(needle, "/") {
			needle = strings.TrimPrefix(needle, "/")
			needle = strings.TrimSuffix(needle, "/")

			if hasAgentMatching(needle, knownAgents) {
				matched++
			} else {
				failed++
			}

			continue
		}

		if hasAgent(needle, knownAgents) {
			matched++
		} else {
			failed++
		}
	}

	return failed == 0 && matched > 0
}

func hasAgentMatching(needle string, stack []string) bool {
	for _, agent := range stack {
		if match, _ := regexp.MatchString(needle, agent); match {
			return true
		}
	}

	return false
}

func hasAgent(needle string, stack []string) bool {
	for _, agent := range stack {
		if agent == needle {
			return true
		}
	}

	return false
}
