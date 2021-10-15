// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EDCH", func() {
	Describe("ECDHKeyPair", func() {
		It("Should create key pairs", func() {
			pub, pri, err := ECDHKeyPair()
			Expect(err).ToNot(HaveOccurred())
			Expect(pub).To(HaveLen(32))
			Expect(pri).To(HaveLen(32))
			Expect(pri).ToNot(Equal(pub))
		})
	})

	Describe("ECDHSharedSecret", func() {
		It("Should correctly calculate secrets", func() {
			alicePri, alicePub, err := ECDHKeyPair()
			Expect(err).ToNot(HaveOccurred())
			Expect(alicePri).ToNot(Equal(alicePub))

			bobPri, bobPub, err := ECDHKeyPair()
			Expect(err).ToNot(HaveOccurred())
			Expect(bobPri).ToNot(Equal(bobPub))
			Expect(bobPri).ToNot(Equal(alicePri))
			Expect(bobPub).ToNot(Equal(alicePub))

			aliceShared, err := ECDHSharedSecret(alicePri, bobPub)
			Expect(err).ToNot(HaveOccurred())
			bobShared, err := ECDHSharedSecret(bobPri, alicePub)
			Expect(err).ToNot(HaveOccurred())
			Expect(aliceShared).To(Equal(bobShared))
			Expect(aliceShared).To(HaveLen(32))

			// fmt.Println()
			// fmt.Printf("Alice Public: %x\n", alicePub)
			// fmt.Printf("Alice Private: %x\n", alicePri)
			// fmt.Printf("Bob Public: %x\n", bobPub)
			// fmt.Printf("Bob Private: %x\n", bobPri)
			// fmt.Printf("Shared: %x\n", bobShared)

			aliceSharedS, err := ECDHSharedSecretString(fmt.Sprintf("%x", alicePri), fmt.Sprintf("%x", bobPub))
			Expect(err).ToNot(HaveOccurred())
			bobSharedS, err := ECDHSharedSecretString(fmt.Sprintf("%x", bobPri), fmt.Sprintf("%x", alicePub))
			Expect(err).ToNot(HaveOccurred())
			Expect(aliceSharedS).To(Equal(bobSharedS))
			Expect(aliceShared).To(HaveLen(32))
		})
	})
})
