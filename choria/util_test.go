// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"time"

	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tokens"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Choria/Util", func() {
	Describe("NatsConnectionHelpers", func() {
		var pk ed25519.PrivateKey
		var pubk ed25519.PublicKey
		var td string
		var err error
		var log *logrus.Entry

		BeforeEach(func() {
			td, err = os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				os.RemoveAll(td)
			})

			pubk, pk, err = iu.Ed25519KeyPairToFile(filepath.Join(td, "seed"))
			Expect(err).ToNot(HaveOccurred())

			log = logrus.NewEntry(logrus.New())
			log.Logger.SetOutput(GinkgoWriter)
		})

		It("Should test required settings", func() {
			_, _, _, err := NatsConnectionHelpers("", "", "", log)
			Expect(err).To(MatchError("collective is required"))

			_, _, _, err = NatsConnectionHelpers("", "choria", "", log)
			Expect(err).To(MatchError("seedfile is required"))
		})

		It("Should fail for unsupported tokens", func() {
			pt, err := tokens.NewProvisioningClaims(true, true, "x", "", "", nil, "example.net", "", "", "choria", "", time.Hour)
			Expect(err).ToNot(HaveOccurred())

			token, err := tokens.SignToken(pt, pk)
			Expect(err).ToNot(HaveOccurred())

			_, _, _, err = NatsConnectionHelpers(token, "choria", filepath.Join(td, "seed"), log)
			Expect(err).To(MatchError("unsupported token purpose: choria_provisioning"))
		})

		It("Should support client tokens", func() {
			ct, err := tokens.NewClientIDClaims("ginkgo", nil, "choria", nil, "", "", time.Hour, nil, pubk)
			Expect(err).ToNot(HaveOccurred())

			token, err := tokens.SignToken(ct, pk)
			Expect(err).ToNot(HaveOccurred())

			inbox, jh, sigh, err := NatsConnectionHelpers(token, "choria", filepath.Join(td, "seed"), log)
			Expect(err).ToNot(HaveOccurred())
			Expect(inbox).To(Equal("choria.reply.4bb6777bb903cae3166e826932f7fe94"))
			Expect(jh()).To(Equal(token))

			expected, err := iu.Ed25519Sign(pk, []byte("toomanysecrets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(sigh([]byte("toomanysecrets"))).To(Equal(expected))
		})

		It("Should support server tokens", func() {
			st, err := tokens.NewServerClaims("ginkgo.example.net", []string{"choria"}, "choria", nil, nil, pubk, "", time.Hour)
			Expect(err).ToNot(HaveOccurred())

			token, err := tokens.SignToken(st, pk)
			Expect(err).ToNot(HaveOccurred())

			inbox, jh, sigh, err := NatsConnectionHelpers(token, "choria", filepath.Join(td, "seed"), log)
			Expect(err).ToNot(HaveOccurred())
			Expect(inbox).To(Equal("choria.reply.3f7c3a791b0eb10da51dca4cdedb9418"))
			Expect(jh()).To(Equal(token))

			expected, err := iu.Ed25519Sign(pk, []byte("toomanysecrets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(sigh([]byte("toomanysecrets"))).To(Equal(expected))
		})
	})
})
