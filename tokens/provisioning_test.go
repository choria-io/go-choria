// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/ed25519"
	"crypto/rand"
	"time"

	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProvisioningClaims", func() {
	var (
		validToken   string
		expiredToken string
		clientToken  string
		pubK         ed25519.PublicKey
		err          error
	)

	BeforeEach(func() {
		pubK, _, err = ed25519.GenerateKey(rand.Reader)
		Expect(err).ToNot(HaveOccurred())

		claims, err := NewClientIDClaims("up=ginkgo", []string{"rpcutil"}, "choria", map[string]string{"group": "admins"}, "// opa policy", "Ginkgo", time.Hour, nil, pubK)
		Expect(err).ToNot(HaveOccurred())
		clientToken, err = SignToken(claims, loadPriKey("testdata/signer-key.pem"))
		Expect(err).ToNot(HaveOccurred())

		pclaims, err := NewProvisioningClaims(true, true, "x", "usr", "toomanysecrets", []string{"nats://example.net:4222"}, "example.net", "/reg.data", "/facts.json", "Ginkgo", time.Hour)
		Expect(err).ToNot(HaveOccurred())
		validToken, err = SignToken(pclaims, loadPriKey("testdata/signer-key.pem"))
		Expect(err).ToNot(HaveOccurred())

		pclaims, err = NewProvisioningClaims(true, true, "x", "usr", "toomanysecrets", []string{"nats://example.net:4222"}, "example.net", "/reg.data", "/facts.json", "Ginkgo", time.Hour)
		Expect(err).ToNot(HaveOccurred())
		pclaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour))
		expiredToken, err = SignToken(pclaims, loadPriKey("testdata/signer-key.pem"))
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewProvisioningClaims", func() {
		It("Should set an issuer if not set", func() {
			claims, err := NewProvisioningClaims(true, true, "x", "usr", "toomanysecrets", []string{"nats://example.net:4222"}, "example.net", "/reg.data", "/facts.json", "", time.Hour)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("Choria"))
		})

		It("Should require url or srv domain", func() {
			claims, err := NewProvisioningClaims(true, true, "x", "usr", "toomanysecrets", nil, "", "/reg.data", "/facts.json", "", time.Hour)
			Expect(err).To(MatchError("srv domain or urls required"))
			Expect(claims).To(BeNil())
		})

		It("Should create correct claims", func() {
			claims, err := NewProvisioningClaims(true, true, "x", "usr", "toomanysecrets", []string{"nats://example.net:4222"}, "example.net", "/reg.data", "/facts.json", "Ginkgo", time.Hour)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("Ginkgo"))
			Expect(claims.Purpose).To(Equal(ProvisioningPurpose))
			Expect(claims.Subject).To(Equal(string(ProvisioningPurpose)))
			Expect(claims.Secure).To(BeTrue())
			Expect(claims.ProvDefault).To(BeTrue())
			Expect(claims.Token).To(Equal("x"))
			Expect(claims.ProvNatsUser).To(Equal("usr"))
			Expect(claims.ProvNatsPass).To(Equal("toomanysecrets"))
			Expect(claims.URLs).To(Equal("nats://example.net:4222"))
			Expect(claims.SRVDomain).To(Equal("example.net"))
			Expect(claims.ProvRegData).To(Equal("/reg.data"))
			Expect(claims.ProvFacts).To(Equal("/facts.json"))
		})
	})

	Describe("IsProvisioningToken", func() {
		It("Should correctly verify", func() {
			Expect(IsProvisioningToken(StandardClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: string(ProvisioningPurpose),
				},
			})).To(BeTrue())

			Expect(IsProvisioningToken(StandardClaims{
				Purpose: ProvisioningPurpose,
			})).To(BeTrue())

			Expect(IsProvisioningToken(StandardClaims{})).To(BeFalse())
		})
	})

	Describe("ParseProvisioningToken", func() {
		It("Should verify the token", func() {
			t, err := ParseProvisioningToken(expiredToken, loadPubKey("testdata/signer-public.pem"))
			Expect(err.Error()).To(MatchRegexp("could not parse provisioner token: token is expired by"))
			Expect(t).To(BeNil())
		})

		It("Should check the purpose", func() {
			t, err := ParseProvisioningToken(clientToken, loadPubKey("testdata/signer-public.pem"))
			Expect(err).To(MatchError("not a provisioning token"))
			Expect(t).To(BeNil())
		})

		It("Should load valid tokens", func() {
			t, err := ParseProvisioningToken(validToken, loadPubKey("testdata/signer-public.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Secure).To(BeTrue())
		})
	})

	Describe("ParseProvisioningTokenWithKeyfile", func() {
		It("Should parse the token", func() {
			t, err := ParseProvisioningTokenWithKeyfile(validToken, "testdata/signer-public.pem")
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Secure).To(BeTrue())
		})
	})

	Describe("ParseProvisionTokenUnverified", func() {
		It("Should only parse prov tokens", func() {
			t, err := ParseProvisionTokenUnverified(clientToken)
			Expect(err).To(MatchError("token is not a provisioning token"))
			Expect(t).To(BeNil())
		})

		It("Should correctly parse", func() {
			t, err := ParseProvisionTokenUnverified(expiredToken)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Secure).To(BeTrue())
		})
	})
})
