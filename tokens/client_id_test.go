// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTokens(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tokens")
}

var _ = Describe("ClientIDClaims", func() {
	var (
		validToken   string
		expiredToken string
		provToken    []byte
		perms        *ClientPermissions
		pubK         ed25519.PublicKey
		err          error
	)

	BeforeEach(func() {
		pubK, _, err = ed25519.GenerateKey(rand.Reader)
		Expect(err).ToNot(HaveOccurred())

		perms = &ClientPermissions{OrgAdmin: true}
		claims, err := NewClientIDClaims("up=ginkgo", []string{"rpcutil"}, "choria", map[string]string{"group": "admins"}, "// opa policy", "Ginkgo", time.Hour, perms, pubK)
		Expect(err).ToNot(HaveOccurred())
		validToken, err = SignToken(claims, loadPriKey("testdata/signer-key.pem"))
		Expect(err).ToNot(HaveOccurred())

		claims, err = NewClientIDClaims("up=ginkgo.expired", []string{"rpcutil"}, "choria", map[string]string{"group": "admins"}, "// opa policy", "Ginkgo", -1*time.Hour, perms, pubK)
		Expect(err).ToNot(HaveOccurred())
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour))
		expiredToken, err = SignToken(claims, loadPriKey("testdata/signer-key.pem"))
		Expect(err).ToNot(HaveOccurred())

		provToken, err = os.ReadFile("testdata/good-provisioning.jwt")
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewClientIDClaims", func() {
		It("Should set an issuer when none is given", func() {
			claims, err := NewClientIDClaims("up=ginkgo", []string{"rpcutil"}, "choria", map[string]string{"group": "admins"}, "// opa policy", "", time.Hour, perms, pubK)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("Choria"))
		})

		It("Should ensure a callerid is set", func() {
			claims, err := NewClientIDClaims("", []string{"rpcutil"}, "choria", map[string]string{"group": "admins"}, "// opa policy", "", time.Hour, perms, pubK)
			Expect(err).To(MatchError("caller id is required"))
			Expect(claims).To(BeNil())
		})

		It("Should create correct claims", func() {
			claims, err := NewClientIDClaims("up=ginkgo", []string{"rpcutil"}, "choria", map[string]string{"group": "admins"}, "// opa policy", "Ginkgo", time.Hour, perms, pubK)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("Ginkgo"))
			Expect(claims.Purpose).To(Equal(ClientIDPurpose))
			Expect(claims.CallerID).To(Equal("up=ginkgo"))
			Expect(claims.AllowedAgents).To(Equal([]string{"rpcutil"}))
			Expect(claims.OrganizationUnit).To(Equal("choria"))
			Expect(claims.UserProperties).To(Equal(map[string]string{"group": "admins"}))
			Expect(claims.OPAPolicy).To(Equal("// opa policy"))
			Expect(claims.PublicKey).To(Equal(hex.EncodeToString(pubK)))
			Expect(claims.IssuedAt.Time).To(BeTemporally("~", time.Now(), time.Second))
			Expect(claims.ExpiresAt.Time).To(BeTemporally("~", time.Now().Add(time.Hour), time.Second))
			Expect(claims.Permissions).To(Equal(perms))
			Expect(claims.Permissions.OrgAdmin).To(BeTrue())
		})
	})

	Describe("ParseClientIDToken", func() {
		It("Should parse any token when not set to validate", func() {
			claims, err := ParseClientIDToken(string(provToken), loadPubKey("testdata/signer-public.pem"), false)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.CallerID).To(Equal(""))

			claims, err = ParseClientIDToken(expiredToken, loadPubKey("testdata/signer-public.pem"), false)
			Expect(err.Error()).To(MatchRegexp("could not parse client id token: token is expired by"))
			Expect(claims).To(BeNil())
		})

		It("Should validate when required", func() {
			claims, err := ParseClientIDToken(expiredToken, loadPubKey("testdata/signer-public.pem"), false)
			Expect(err.Error()).To(MatchRegexp("could not parse client id token: token is expired by"))
			Expect(claims).To(BeNil())

			claims, err = ParseClientIDToken(string(provToken), loadPubKey("testdata/signer-public.pem"), true)
			Expect(err.Error()).To(MatchRegexp("not a client id token"))
			Expect(claims).To(BeNil())

			claims, err = ParseClientIDToken(validToken, loadPubKey("testdata/signer-public.pem"), true)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.CallerID).To(Equal("up=ginkgo"))
		})
	})

	Describe("ParseClientIDTokenWithKeyfile", func() {
		It("Should parse using the file", func() {
			claims, err := ParseClientIDTokenWithKeyfile(validToken, "testdata/signer-public.pem", false)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.CallerID).To(Equal("up=ginkgo"))
		})
	})

	Describe("IsClientIDToken", func() {
		It("Should correctly check", func() {
			Expect(IsClientIDTokenString(validToken)).To(BeTrue())
			Expect(IsClientIDTokenString(expiredToken)).To(BeTrue())
			Expect(IsClientIDTokenString(string(provToken))).To(BeFalse())
		})
	})

	Describe("UnverifiedCallerFromClientIDToken", func() {
		It("Should parse tokens", func() {
			t, caller, err := UnverifiedCallerFromClientIDToken(expiredToken)
			Expect(err).ToNot(HaveOccurred())
			Expect(caller).To(Equal("up=ginkgo.expired"))
			Expect(t.Valid).To(BeFalse())

			t, caller, err = UnverifiedCallerFromClientIDToken(validToken)
			Expect(err).ToNot(HaveOccurred())
			Expect(caller).To(Equal("up=ginkgo"))
			Expect(t.Valid).To(BeFalse())
		})

		It("Should fail for wrong tokens", func() {
			t, caller, err := UnverifiedCallerFromClientIDToken(string(provToken))
			Expect(err).To(MatchError("not a client id token"))
			Expect(caller).To(Equal(""))
			Expect(t).To(BeNil())
		})
	})
})
