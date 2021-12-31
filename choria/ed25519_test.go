// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"encoding/hex"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ed25519", func() {
	Describe("Ed25519KeyPair", func() {
		It("Should generate a matching keypair", func() {
			pub, pri, err := Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())
			Expect(pub.Equal(pri.Public())).To(BeTrue())
		})
	})

	Describe("Seed Files", func() {
		It("Should store the seed to a file", func() {
			td, err := os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(td)

			seedFile := filepath.Join(td, "key.seed")
			pub, pri, err := Ed25519KeyPairToFile(seedFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(pub.Equal(pri.Public())).To(BeTrue())

			npub, npri, err := Ed25519KeyPairFromSeedFile(seedFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(npub.Equal(npri.Public())).To(BeTrue())
			Expect(npub.Equal(pri.Public())).To(BeTrue())
			Expect(pub.Equal(npri.Public())).To(BeTrue())
			Expect(pub.Equal(npub)).To(BeTrue())

			s, err := os.ReadFile(seedFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(s)).To(Equal(hex.EncodeToString(pri.Seed())))

			nseed, err := hex.DecodeString(string(s))
			Expect(err).ToNot(HaveOccurred())
			npub, npri, err = Ed25519KeyPairFromSeed(nseed)
			Expect(err).ToNot(HaveOccurred())
			Expect(npub.Equal(npri.Public())).To(BeTrue())
			Expect(npub.Equal(pri.Public())).To(BeTrue())
			Expect(pub.Equal(npri.Public())).To(BeTrue())
			Expect(pub.Equal(npub)).To(BeTrue())
		})
	})

	Describe("Signing", func() {
		It("Should make correct signatures", func() {
			td, err := os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(td)

			seed, err := hex.DecodeString("8e306060341f7eb867c7d09609d53bfa9e6cb38ca744c0dca548572cc3080b6a")
			Expect(err).ToNot(HaveOccurred())
			_, pri, err := Ed25519KeyPairFromSeed(seed)
			Expect(err).ToNot(HaveOccurred())

			seedFile := filepath.Join(td, "key.seed")
			err = os.WriteFile(seedFile, []byte(hex.EncodeToString(seed)), 0600)
			Expect(err).ToNot(HaveOccurred())

			sig, err := Ed25519Sign(pri, []byte("too many secrets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(hex.EncodeToString(sig)).To(Equal("5971db5ce8eec72d586b0630e2cdd9464e6800b973e6c58575a4072018ca51a93f2e1988d47e058bb19c18d57a44ffa9931b6b7e2f70b5e44ddc50339a8c790b"))

			sig, err = Ed25519SignWithSeedFile(seedFile, []byte("too many secrets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(hex.EncodeToString(sig)).To(Equal("5971db5ce8eec72d586b0630e2cdd9464e6800b973e6c58575a4072018ca51a93f2e1988d47e058bb19c18d57a44ffa9931b6b7e2f70b5e44ddc50339a8c790b"))
		})
	})
})
