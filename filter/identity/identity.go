// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"regexp"
	"strings"
)

// Match identities on a OR basis, since nodes have only 1 identity
func Match(filters []string, certname string) bool {
	for _, filter := range filters {
		if match(certname, filter) {
			return true
		}
	}

	return false
}

// FilterNodes return only nodes matching f
func FilterNodes(nodes []string, f string) []string {
	matched := []string{}

	for _, n := range nodes {
		if match(n, f) {
			matched = append(matched, n)
		}
	}

	return matched
}

func match(certname string, filter string) bool {
	if strings.HasPrefix(filter, "/") && strings.HasSuffix(filter, "/") {
		filter = strings.TrimPrefix(filter, "/")
		filter = strings.TrimSuffix(filter, "/")
		if matched, _ := regexp.MatchString(filter, certname); matched {
			return true
		}
	}

	return certname == filter
}
