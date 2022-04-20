// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"context"
	"os"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/tokens"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Connector", func() {
	var cfg *config.Config

	BeforeEach(func() {
		skipConnect = true

		cfg = config.NewConfigForTests()
		cfg.DisableTLS = true
		cfg.Choria.SSLDir = "/nonexisting"
	})

	Describe("NewConnector", func() {
		genToken := func(id string) string {
			tf, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())

			pk, _, err := Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())

			t, err := tokens.NewServerClaims(id, []string{"choria"}, "choria", nil, []string{}, pk, "ginkgo", time.Hour)
			Expect(err).ToNot(HaveOccurred())
			s, err := tokens.SignTokenWithKeyFile(t, "../tokens/testdata/signer-key.pem")
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
