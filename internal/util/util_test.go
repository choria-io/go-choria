// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGinkgo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal/Util")
}

var _ = Describe("Internal/Util", func() {
	Describe("HasPrefix", func() {
		It("Should function correctly", func() {
			Expect(HasPrefix("foo.bar", "bar", ".bar", "f", "meh")).To(BeTrue())
			Expect(HasPrefix("foo.bar", "foo", "bar", ".bar", "f", "meh")).To(BeTrue())
			Expect(HasPrefix("foo.bar", "!foo", "bar", ".bar", "xf")).To(BeFalse())
		})
	})

	Describe("Sha256Bytes", func() {
		It("Should correctly calculate the checksum", func() {
			Expect(Sha256HashBytes([]byte("sample file"))).To(Equal("9f28ca60126cb0c438bc90f6d323efb4abf699f976c18a7a88cdb166e45e22ec"))
		})
	})

	Describe("Sha256HashFile", func() {
		It("Should handle errors", func() {
			s, err := Sha256HashFile("/nonexisting")
			Expect(err).To(HaveOccurred())
			Expect(s).To(Equal(""))
		})

		It("Should match expected sum", func() {
			Expect(Sha256HashFile("testdata/9f28c.txt")).To(Equal("9f28ca60126cb0c438bc90f6d323efb4abf699f976c18a7a88cdb166e45e22ec"))
		})
	})

	Describe("FileHasSha256Sum", func() {
		It("Should handle errors", func() {
			ok, s, err := FileHasSha256Sum("/nonexisting", "x")
			Expect(err).To(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(s).To(Equal(""))
		})

		It("Should correctly handle invalid hashes", func() {
			ok, s, err := FileHasSha256Sum("testdata/9f28c.txt", "x")
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(s).To(Equal("9f28ca60126cb0c438bc90f6d323efb4abf699f976c18a7a88cdb166e45e22ec"))
		})

		It("Should support valid hashes", func() {
			ok, s, err := FileHasSha256Sum("testdata/9f28c.txt", "9f28ca60126cb0c438bc90f6d323efb4abf699f976c18a7a88cdb166e45e22ec")
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(s).To(Equal("9f28ca60126cb0c438bc90f6d323efb4abf699f976c18a7a88cdb166e45e22ec"))
		})
	})
})
