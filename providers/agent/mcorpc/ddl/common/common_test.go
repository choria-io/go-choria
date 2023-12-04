// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Agent/McoRPC/DDL/Common")
}

var _ = Describe("Providers/McoRPC/DDL/Common", func() {
	Describe("EachFile", func() {
		It("Should call the cb with the right files", func() {
			files := make(map[string]string)

			EachFile("agent", []string{"nonexisting", filepath.Join("..", "agent", "testdata")}, func(n, p string) bool {
				files[n] = p
				return false
			})

			Expect(files["package"]).To(Equal(filepath.Join("..", "agent", "testdata", "mcollective", "agent", "package.json")))
		})
	})

	Describe("validateStringValidation", func() {
		It("Should support shellsafe", func() {
			w, err := validateStringValidation("shellsafe", ">")
			Expect(w).To(BeEmpty())
			Expect(err).To(MatchError("may not contain '>'"))

			w, err = validateStringValidation("shellsafe", "foo")
			Expect(w).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support ipv4address", func() {
			w, err := validateStringValidation("ipv4address", "2a00:1450:4002:807::200e")
			Expect(w).To(BeEmpty())
			Expect(err).To(MatchError("2a00:1450:4002:807::200e is not an IPv4 address"))

			w, err = validateStringValidation("ipv4address", "1.1.1.1")
			Expect(w).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support ipv6address", func() {
			w, err := validateStringValidation("ipv6address", "2a00:1450:4002:807::200e")
			Expect(w).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())

			w, err = validateStringValidation("ipv6address", "1.1.1.1")
			Expect(w).To(BeEmpty())
			Expect(err).To(MatchError("1.1.1.1 is not an IPv6 address"))
		})

		It("Should support ipaddress", func() {
			w, err := validateStringValidation("ipaddress", "2a00:1450:4002:807::200e")
			Expect(w).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())

			w, err = validateStringValidation("ipaddress", "bob")
			Expect(w).To(BeEmpty())
			Expect(err).To(MatchError("bob is not an IP address"))
		})

		It("Should warn but not fail for validators that start with alpha characters", func() {
			w, err := validateStringValidation("foo", "2a00:1450:4002:807::200e")
			Expect(w).To(HaveLen(1))
			Expect(w[0]).To(Equal("Unsupported validator 'foo'"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support regex validators", func() {
			w, err := validateStringValidation("^2a00", "2a00:1450:4002:807::200e")
			Expect(w).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())

			w, err = validateStringValidation("\\d+", "bob")
			Expect(w).To(BeEmpty())
			Expect(err).To(MatchError("input does not match '\\d+'"))
		})
	})
})
