// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"context"
	"crypto/fips140"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/choria-io/go-choria/config"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/tokens"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Connector", func() {
	var cfg *config.Config

	BeforeEach(func() {
		skipConnect = true

		cfg = config.NewConfigForTests()
		cfg.DisableTLS = true
		cfg.Choria.SSLDir = "/nonexisting"
	})

	Describe("ReplyTarget", func() {
		It("Should produce the correct reply target", func() {
			mockctl := gomock.NewController(GinkgoT())
			msg := imock.NewMockMessage(mockctl)
			msg.EXPECT().Collective().Return("mcollective").AnyTimes()
			msg.EXPECT().CallerID().Return("choria=rip.mcollective").AnyTimes()

			result := ReplyTarget(msg, "abc123")

			parts := strings.Split(result, ".")
			Expect(parts).To(HaveLen(4))
			Expect(parts[0]).To(Equal("mcollective"))
			Expect(parts[1]).To(Equal("reply"))
			Expect(parts[3]).To(Equal("abc123"))

			if fips140.Enabled() {
				expected := fmt.Sprintf("%x", sha256.Sum256([]byte("choria=rip.mcollective")))
				Expect(parts[2]).To(Equal(expected))
			} else {
				expected := fmt.Sprintf("%x", md5.Sum([]byte("choria=rip.mcollective")))
				Expect(parts[2]).To(Equal(expected))
			}
		})
	})

	Describe("Inbox", func() {
		It("Should produce the correct inbox", func() {
			result := Inbox("mcollective", "choria=rip.mcollective")

			parts := strings.Split(result, ".")
			Expect(parts).To(HaveLen(4))
			Expect(parts[0]).To(Equal("mcollective"))
			Expect(parts[1]).To(Equal("reply"))

			if fips140.Enabled() {
				expected := fmt.Sprintf("%x", sha256.Sum256([]byte("choria=rip.mcollective")))
				Expect(parts[2]).To(Equal(expected))
			} else {
				expected := fmt.Sprintf("%x", md5.Sum([]byte("choria=rip.mcollective")))
				Expect(parts[2]).To(Equal(expected))
			}

			// last part is a unique ID, just check it's not empty
			Expect(parts[3]).ToNot(BeEmpty())
		})
	})

	Describe("NewConnector", func() {
		genToken := func(id string) string {
			tf, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())

			pk, _, err := Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())

			t, err := tokens.NewServerClaims(id, []string{"choria"}, "choria", nil, []string{}, pk, "ginkgo", time.Hour)
			Expect(err).ToNot(HaveOccurred())
			s, err := tokens.SignTokenWithKeyFile(t, "testdata/rsa/signer-key.pem")
			Expect(err).ToNot(HaveOccurred())

			_, err = tf.WriteString(s)
			Expect(err).ToNot(HaveOccurred())
			tf.Close()

			return tf.Name()
		}

		It("Should fail to connect when a JWT token does not match the identity", func() {
			cfg.InitiatedByServer = true
			cfg.Choria.ServerAnonTLS = true
			fw, err := NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			t := genToken("other.example.net")
			defer os.RemoveAll(t)
			cfg.Choria.ServerTokenFile = t

			conn, err := fw.NewConnector(context.Background(), fw.MiddlewareServers, "ginkgo", fw.Logger("ginkgo"))
			Expect(err).To(MatchError("identity ginkgo.example.net does not match caller other.example.net in JWT token"))
			Expect(conn).To(BeNil())
		})

		It("Should connect with a JWT that match the identity", func() {
			cfg.InitiatedByServer = true
			cfg.Choria.ServerAnonTLS = true
			fw, err := NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			t := genToken(cfg.Identity)
			defer os.RemoveAll(t)

			cfg.Choria.ServerTokenFile = t

			conn, err := fw.NewConnector(context.Background(), fw.MiddlewareServers, "ginkgo", fw.Logger("ginkgo"))
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
		})
	})
})
