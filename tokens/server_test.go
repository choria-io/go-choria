// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServerClaims", func() {
	var (
		pubK ed25519.PublicKey
		priK ed25519.PrivateKey
		err  error
	)

	BeforeEach(func() {
		pubK, priK, err = ed25519.GenerateKey(rand.Reader)
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

	Describe("IsMatchingPublicKey", func() {
		var claims *ServerClaims
		var err error

		BeforeEach(func() {
			perms := &ServerPermissions{Submission: true}
			claims, err = NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, []string{"choria.registration"}, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should detect incorrect length public keys", func() {
			match, err := claims.IsMatchingPublicKey(nil)
			Expect(match).To(BeFalse())
			Expect(err).To(MatchError("invalid size for public key"))

			claims.PublicKey = claims.PublicKey[2:]
			match, err = claims.IsMatchingPublicKey(pubK)
			Expect(match).To(BeFalse())
			Expect(err).To(MatchError("invalid size for token stored public key"))

			claims.PublicKey = ""
			match, err = claims.IsMatchingPublicKey(pubK)
			Expect(match).To(BeFalse())
			Expect(err).To(MatchError("no public key stored in the JWT"))
		})

		It("Should fail for invalid public keys", func() {
			pK, _, err := ed25519.GenerateKey(rand.Reader)
			Expect(err).ToNot(HaveOccurred())
			match, err := claims.IsMatchingPublicKey(pK)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(BeFalse())
		})

		It("Should match correct keys", func() {
			match, err := claims.IsMatchingPublicKey(pubK)
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(BeTrue())
		})
	})

	Describe("IsMatchingSeedFile", func() {
		var claims *ServerClaims
		var err error

		BeforeEach(func() {
			perms := &ServerPermissions{Submission: true}
			claims, err = NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, []string{"choria.registration"}, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail for invalid files", func() {
			match, err := claims.IsMatchingSeedFile("/nonexisting")
			Expect(match).To(BeFalse())
			if runtime.GOOS == "windows" {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).To(MatchError("open /nonexisting: no such file or directory"))
			}

			tf, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tf.Name())
			tf.WriteString("x")
			tf.Close()

			match, err = claims.IsMatchingSeedFile(tf.Name())
			Expect(match).To(BeFalse())
			Expect(err).To(HaveOccurred())
		})

		It("Should fail for invalid seeds", func() {
			_, priK, err := ed25519.GenerateKey(rand.Reader)
			Expect(err).ToNot(HaveOccurred())
			tf, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tf.Name())
			_, err = tf.Write([]byte(hex.EncodeToString(priK.Seed())))
			Expect(err).ToNot(HaveOccurred())
			tf.Close()

			match, err := claims.IsMatchingSeedFile(tf.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(BeFalse())
		})

		It("Should succeed for correct seeds", func() {
			tf, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tf.Name())
			_, err = tf.Write([]byte(hex.EncodeToString(priK.Seed())))
			Expect(err).ToNot(HaveOccurred())
			tf.Close()

			match, err := claims.IsMatchingSeedFile(tf.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(match).To(BeTrue())
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
		It("Should fail for invalid rsa tokens", func() {
			perms := &ServerPermissions{Submission: true}
			claims, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, nil, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
			signed, err := SignTokenWithKeyFile(claims, "testdata/rsa/signer-key.pem")
			Expect(err).ToNot(HaveOccurred())

			_, err = ParseServerTokenWithKeyfile(signed, "testdata/rsa/other-public.pem")
			Expect(err).To(MatchError("could not parse server id token: crypto/rsa: verification error"))
		})

		It("Should fail for invalid ed25519 tokens", func() {
			perms := &ServerPermissions{Submission: true}
			claims, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, nil, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
			signed, err := SignTokenWithKeyFile(claims, "testdata/ed25519/signer.seed")
			Expect(err).ToNot(HaveOccurred())

			_, err = ParseServerTokenWithKeyfile(signed, "testdata/ed25519/other.public")
			Expect(err).To(MatchError("could not parse server id token: ed25519: verification error"))
		})

		It("Should parse valid rsa tokens", func() {
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

		It("Should parse valid ed25519 tokens", func() {
			perms := &ServerPermissions{Submission: true}
			claims, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", perms, nil, pubK, "ginkgo issuer", 365*24*time.Hour)
			Expect(err).ToNot(HaveOccurred())
			signed, err := SignTokenWithKeyFile(claims, "testdata/ed25519/signer.seed")
			Expect(err).ToNot(HaveOccurred())

			_, err = ParseServerTokenWithKeyfile(signed, "testdata/ed25519/signer.public")
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.ChoriaIdentity).To(Equal("ginkgo.example.net"))
		})
	})
})
