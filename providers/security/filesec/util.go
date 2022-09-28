// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package filesec

import (
	"os"
	"regexp"
	"runtime"
	"strings"
)

// MatchAnyRegex checks str against a list of possible regex, if any match true is returned
func MatchAnyRegex(str string, regex []string) bool {
	matcher := regexp.MustCompile("^/.+/$")

	for _, reg := range regex {
		if matcher.MatchString(reg) {
			reg = strings.TrimLeft(reg, "/")
			reg = strings.TrimRight(reg, "/")
		}

		if matched, _ := regexp.MatchString(reg, str); matched {
			return true
		}
	}

	return false
}

func uid() int {
	if useFakeUID {
		return fakeUID
	}

	return os.Geteuid()
}

func runtimeOs() string {
	if useFakeOS {
		return fakeOS
	}

	return runtime.GOOS
}
