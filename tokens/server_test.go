// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServerClaims", func() {
	var (
		pubK ed25519.PublicKey
		err  error
	)

	BeforeEach(func() {
		pubK, _, err = ed25519.GenerateKey(rand.Reader)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewServerClaims", func() {
		It("Should require identity", func() {
			_, err := NewServerClaims("", nil, "", nil, nil, nil, "", 0)
			Expect(err).To(MatchError("identity is required"))
		})

		It("Should require collectives", func() {
			_, err := NewServerClaims("ginkgo.example.net", nil, "", nil, nil, nil, "", 0)
			Expect(err).To(MatchError("at least one collective is required"))
		})

		It("Should require public key", func() {
			_, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "", nil, nil, nil, "", 0)
			Expect(err).To(MatchError("public key is required"))
		})

		It("Should create a valid token", func() {
			perms := &ServerPermissions{Submission: true}
			claims, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, []string{"choria.registration"}, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.ChoriaIdentity).To(Equal("ginkgo.example.net"))
			Expect(claims.Purpose).To(Equal(ServerPurpose))
			Expect(claims.Permissions.Submission).To(BeTrue())
			Expect(claims.Collectives).To(Equal([]string{"choria"}))
			Expect(claims.PublicKey).To(Equal(hex.EncodeToString(pubK)))
			Expect(claims.OrganizationUnit).To(Equal("ginkgo_org"))
			Expect(claims.Issuer).To(Equal("ginkgo issuer"))
			Expect(claims.AdditionalPublishSubjects).To(Equal([]string{"choria.registration"}))
			Expect(claims.IssuedAt.Time).To(BeTemporally("~", time.Now(), time.Second))
			Expect(claims.ExpiresAt.Time).To(BeTemporally("~", time.Now().Add(365*24*time.Hour), time.Second))
		})
	})

	Describe("IsServerTokenString", func() {
		It("Should detect correctly", func() {
			pt, err := os.ReadFile("testdata/rsa/good-provisioning.jwt")
			Expect(err).ToNot(HaveOccurred())
			Expect(IsServerTokenString(string(pt))).To(BeFalse())

			perms := &ServerPermissions{Submission: true}
			claims, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, nil, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
			signed, err := SignTokenWithKeyFile(claims, "testdata/rsa/signer-key.pem")
			Expect(err).ToNot(HaveOccurred())

			Expect(IsServerTokenString(signed)).To(BeTrue())
		})
	})

	Describe("IsServerToken", func() {
		It("Should detect correctly", func() {
			Expect(IsServerToken(StandardClaims{})).To(BeFalse())
			Expect(IsServerToken(StandardClaims{Purpose: ServerPurpose})).To(BeTrue())
		})
	})

	Describe("ParseServerTokenUnverified", func() {
		It("Should fail for wrong kinds of tokens", func() {
			pt, err := os.ReadFile("testdata/rsa/good-provisioning.jwt")
			Expect(err).ToNot(HaveOccurred())
			_, err = ParseServerTokenUnverified(string(pt))
			Expect(err).To(MatchError("not a server token"))
		})

		It("Should parse valid tokens", func() {
			perms := &ServerPermissions{Submission: true}
			claims, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, nil, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
			signed, err := SignTokenWithKeyFile(claims, "testdata/rsa/signer-key.pem")
			Expect(err).ToNot(HaveOccurred())

			claims = nil
			claims, err = ParseServerTokenUnverified(signed)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.ChoriaIdentity).To(Equal("ginkgo.example.net"))
		})
	})

	Describe("ParseServerTokenWithKeyfile", func() {
		It("Should fail for invalid tokens", func() {
			perms := &ServerPermissions{Submission: true}
			claims, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, nil, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
			signed, err := SignTokenWithKeyFile(claims, "testdata/rsa/signer-key.pem")
			Expect(err).ToNot(HaveOccurred())

			_, err = ParseServerTokenWithKeyfile(signed, "testdata/rsa/other-public.pem")
			Expect(err).To(MatchError("could not parse server id token: crypto/rsa: verification error"))
		})

		It("Should parse valid token", func() {
			perms := &ServerPermissions{Submission: true}
			claims, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, []string{"additional.subject"}, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
			signed, err := SignTokenWithKeyFile(claims, "testdata/rsa/signer-key.pem")
			Expect(err).ToNot(HaveOccurred())

			claims = nil
			claims, err = ParseServerTokenWithKeyfile(signed, "testdata/rsa/signer-public.pem")
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.ChoriaIdentity).To(Equal("ginkgo.example.net"))
		})
	})
})
