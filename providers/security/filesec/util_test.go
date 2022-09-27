// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package filesec

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MatchAnyRegex", func() {
	It("Should correctly match valid patterns", func() {
		patterns := []string{
			"bare",
			"/this.+other/",
		}

		Expect(MatchAnyRegex("this is a bare word sentence", patterns)).To(BeTrue())
		Expect(MatchAnyRegex("this, that and the other", patterns)).To(BeTrue())
		Expect(MatchAnyRegex("no match", patterns)).To(BeFalse())
	})
})
