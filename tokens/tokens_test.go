// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/rsa"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func loadPubKey(k string) *rsa.PublicKey {
	f, err := os.ReadFile(k)
	Expect(err).ToNot(HaveOccurred())
	pubK, err := jwt.ParseRSAPublicKeyFromPEM(f)
	Expect(err).ToNot(HaveOccurred())

	return pubK
}

func loadPriKey(k string) *rsa.PrivateKey {
	f, err := os.ReadFile(k)
	Expect(err).ToNot(HaveOccurred())
	pK, err := jwt.ParseRSAPrivateKeyFromPEM(f)
	Expect(err).ToNot(HaveOccurred())
	return pK
}

var _ = Describe("Tokens", func() {
	var (
		provJWT []byte
		err     error
	)

	BeforeEach(func() {
		provJWT, err = os.ReadFile("testdata/good-provisioning.jwt")
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("ParseToken", func() {
		It("Should parse and verify the token", func() {
			claims := &jwt.MapClaims{}

			err = ParseToken(string(provJWT), claims, nil)
			Expect(err).To(MatchError("invalid public key"))

			err = ParseToken(string(provJWT), claims, loadPubKey("testdata/other-public.pem"))
			Expect(err).To(MatchError("crypto/rsa: verification error"))

			err = ParseToken(string(provJWT), claims, loadPubKey("testdata/signer-public.pem"))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TokenPurpose", func() {
		It("Should extract the correct purpose", func() {
			Expect(TokenPurposeBytes(provJWT)).To(Equal(ProvisioningPurpose))
			Expect(TokenPurpose(string(provJWT))).To(Equal(ProvisioningPurpose))
		})
	})

	Describe("SignToken", func() {
		It("Should correctly sign the token", func() {
			claims, err := newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
			Expect(err).ToNot(HaveOccurred())
			t, err := SignToken(claims, loadPriKey("testdata/signer-key.pem"))
			Expect(err).ToNot(HaveOccurred())

			claims = &StandardClaims{}
			err = ParseToken(t, claims, loadPubKey("testdata/signer-public.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("ginkgo"))
		})
	})

	Describe("SignTokenWithKeyFile", func() {
		It("Should correctly sign the token", func() {
			claims, err := newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
			Expect(err).ToNot(HaveOccurred())
			t, err := SignTokenWithKeyFile(claims, "testdata/signer-key.pem")
			Expect(err).ToNot(HaveOccurred())

			claims = &StandardClaims{}
			err = ParseToken(t, claims, loadPubKey("testdata/signer-public.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("ginkgo"))
		})
	})

	Describe("SaveAndSignTokenWithKeyFile", func() {
		It("Should correctly sign and save", func() {
			td, err := os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(td)

			out := filepath.Join(td, "token.jwt")
			claims, err := newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
			Expect(err).ToNot(HaveOccurred())
			err = SaveAndSignTokenWithKeyFile(claims, "testdata/signer-key.pem", out, 0600)
			Expect(err).ToNot(HaveOccurred())

			stat, err := os.Stat(out)
			Expect(err).ToNot(HaveOccurred())
			if runtime.GOOS == "windows" {
				Expect(stat.Mode()).To(Equal(os.FileMode(0666)))
			} else {
				Expect(stat.Mode()).To(Equal(os.FileMode(0600)))
			}
			t, err := os.ReadFile(out)
			Expect(err).ToNot(HaveOccurred())
			claims = &StandardClaims{}
			err = ParseToken(string(t), claims, loadPubKey("testdata/signer-public.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("ginkgo"))
		})
	})

	Describe("newStandardClaims", func() {
		It("Should create correct claims", func() {
			claims, err := newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("ginkgo"))
			Expect(claims.Purpose).To(Equal(ProvisioningPurpose))
			Expect(claims.IssuedAt.Time).To(BeTemporally("~", time.Now(), time.Second))
			Expect(claims.ExpiresAt).To(BeNil())

			claims, err = newStandardClaims("ginkgo", ProvisioningPurpose, time.Hour, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("ginkgo"))
			Expect(claims.Purpose).To(Equal(ProvisioningPurpose))
			Expect(claims.Subject).To(Equal(string(ProvisioningPurpose)))
			Expect(claims.IssuedAt.Time).To(BeTemporally("~", time.Now(), time.Second))
			Expect(claims.ExpiresAt.Time).To(BeTemporally("~", time.Now().Add(time.Hour), time.Second))
		})
	})
})
