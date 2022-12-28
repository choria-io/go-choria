// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func loadRSAPubKey(k string) *rsa.PublicKey {
	f, err := os.ReadFile(k)
	Expect(err).ToNot(HaveOccurred())
	pubK, err := jwt.ParseRSAPublicKeyFromPEM(f)
	Expect(err).ToNot(HaveOccurred())

	return pubK
}

func loadEd25519Seed(f string) (ed25519.PublicKey, ed25519.PrivateKey) {
	ss, err := os.ReadFile(f)
	Expect(err).ToNot(HaveOccurred())

	seed, err := hex.DecodeString(string(ss))
	Expect(err).ToNot(HaveOccurred())

	priK := ed25519.NewKeyFromSeed(seed)
	pubK := priK.Public().(ed25519.PublicKey)
	return pubK, priK
}

func loadRSAPriKey(k string) *rsa.PrivateKey {
	f, err := os.ReadFile(k)
	Expect(err).ToNot(HaveOccurred())
	pK, err := jwt.ParseRSAPrivateKeyFromPEM(f)
	Expect(err).ToNot(HaveOccurred())
	return pK
}

var _ = Describe("Tokens", func() {
	var (
		provJWTRSA     []byte
		provJWTED25519 []byte
		err            error
	)

	BeforeEach(func() {
		provJWTRSA, err = os.ReadFile("testdata/rsa/good-provisioning.jwt")
		Expect(err).ToNot(HaveOccurred())
		provJWTED25519, err = os.ReadFile("testdata/ed25519/good-provisioning.jwt")
		Expect(err).ToNot(HaveOccurred())

	})

	Describe("ParseToken", func() {
		Describe("ED25519", func() {
			It("Should parse and verify the token", func() {
				claims := &jwt.MapClaims{}
				err = ParseToken(string(provJWTED25519), claims, nil)
				Expect(err).To(MatchError("invalid public key"))

				pubK, _ := loadEd25519Seed("testdata/ed25519/other.seed")
				err = ParseToken(string(provJWTED25519), claims, pubK)
				Expect(err).To(MatchError("ed25519: verification error"))

				err = ParseToken(string(provJWTED25519), claims, loadRSAPubKey("testdata/rsa/other-public.pem"))
				Expect(err).To(MatchError("ed25519 public key required"))

				pubK, _ = loadEd25519Seed("testdata/ed25519/signer.seed")
				err = ParseToken(string(provJWTED25519), claims, pubK)
				Expect(err).ToNot(HaveOccurred())

				sclaims, err := NewServerClaims("ginkgo.example.net", []string{"choria"}, "ginkgo_org", nil, nil, pubK, "ginkgo issuer", 365*24*time.Hour)
				Expect(err).ToNot(HaveOccurred())
				signed, err := SignTokenWithKeyFile(sclaims, "testdata/ed25519/signer.seed")
				Expect(err).ToNot(HaveOccurred())

				err = ParseToken(signed, claims, pubK)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("RSA", func() {
			It("Should parse and verify the token", func() {
				claims := &jwt.MapClaims{}

				err = ParseToken(string(provJWTRSA), claims, nil)
				Expect(err).To(MatchError("invalid public key"))

				err = ParseToken(string(provJWTRSA), claims, loadRSAPubKey("testdata/rsa/other-public.pem"))
				Expect(err).To(MatchError("crypto/rsa: verification error"))

				pubK, _ := loadEd25519Seed("testdata/ed25519/other.seed")
				err = ParseToken(string(provJWTRSA), claims, pubK)
				Expect(err).To(MatchError("rsa public key required"))

				err = ParseToken(string(provJWTRSA), claims, loadRSAPubKey("testdata/rsa/signer-public.pem"))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("TokenPurpose", func() {
		It("Should extract the correct purpose", func() {
			Expect(TokenPurposeBytes(provJWTRSA)).To(Equal(ProvisioningPurpose))
			Expect(TokenPurpose(string(provJWTRSA))).To(Equal(ProvisioningPurpose))
		})
	})

	Describe("SignToken", func() {
		Describe("ED25519", func() {
			It("Should correctly sign the token", func() {
				claims, err := newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
				Expect(err).ToNot(HaveOccurred())

				pubK, priK := loadEd25519Seed("testdata/ed25519/signer.seed")

				t, err := SignToken(claims, priK)
				Expect(err).ToNot(HaveOccurred())

				claims = &StandardClaims{}
				err = ParseToken(t, claims, loadRSAPubKey("testdata/rsa/signer-public.pem"))
				Expect(err).To(MatchError("ed25519 public key required"))

				claims = &StandardClaims{}
				err = ParseToken(t, claims, pubK)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.Issuer).To(Equal("ginkgo"))
			})
		})

		Describe("RSA", func() {
			It("Should correctly sign the token", func() {
				claims, err := newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
				Expect(err).ToNot(HaveOccurred())
				t, err := SignToken(claims, loadRSAPriKey("testdata/rsa/signer-key.pem"))
				Expect(err).ToNot(HaveOccurred())

				pubK, _ := loadEd25519Seed("testdata/ed25519/signer.seed")

				claims = &StandardClaims{}
				err = ParseToken(t, claims, pubK)
				Expect(err).To(MatchError("rsa public key required"))

				err = ParseToken(t, claims, loadRSAPubKey("testdata/rsa/signer-public.pem"))
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.Issuer).To(Equal("ginkgo"))
			})
		})
	})

	Describe("SignTokenWithKeyFile", func() {
		Describe("ED25519", func() {
			It("Should correctly sign the token", func() {
				claims, err := newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
				Expect(err).ToNot(HaveOccurred())

				t, err := SignTokenWithKeyFile(claims, "testdata/ed25519/signer.seed")
				Expect(err).ToNot(HaveOccurred())

				pubK, _ := loadEd25519Seed("testdata/ed25519/signer.seed")
				claims = &StandardClaims{}
				err = ParseToken(t, claims, pubK)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.Issuer).To(Equal("ginkgo"))
			})
		})

		Describe("RSA", func() {
			It("Should correctly sign the token", func() {
				claims, err := newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
				Expect(err).ToNot(HaveOccurred())
				t, err := SignTokenWithKeyFile(claims, "testdata/rsa/signer-key.pem")
				Expect(err).ToNot(HaveOccurred())

				claims = &StandardClaims{}
				err = ParseToken(t, claims, loadRSAPubKey("testdata/rsa/signer-public.pem"))
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.Issuer).To(Equal("ginkgo"))
			})
		})
	})

	Describe("SaveAndSignTokenWithKeyFile", func() {
		var td string
		var err error
		var claims *StandardClaims
		var out string

		BeforeEach(func() {
			td, err = os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			out = filepath.Join(td, "token.jwt")
			claims, err = newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(td)
		})

		Describe("ED25519", func() {
			It("Should correctly sign and save", func() {
				err = SaveAndSignTokenWithKeyFile(claims, "testdata/ed25519/signer.seed", out, 0600)
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
				pubK, _ := loadEd25519Seed("testdata/ed25519/signer.seed")
				err = ParseToken(string(t), claims, pubK)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.Issuer).To(Equal("ginkgo"))
			})
		})

		Describe("RSA", func() {
			It("Should correctly sign and save", func() {
				err = SaveAndSignTokenWithKeyFile(claims, "testdata/rsa/signer-key.pem", out, 0600)
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
				err = ParseToken(string(t), claims, loadRSAPubKey("testdata/rsa/signer-public.pem"))
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.Issuer).To(Equal("ginkgo"))
			})
		})
	})

	Describe("newStandardClaims", func() {
		It("Should create correct claims", func() {
			claims, err := newStandardClaims("ginkgo", ProvisioningPurpose, 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("ginkgo"))
			Expect(claims.Purpose).To(Equal(ProvisioningPurpose))
			Expect(claims.IssuedAt.Time).To(BeTemporally("~", time.Now(), time.Second))
			Expect(claims.ExpiresAt.Time).To(BeTemporally("~", time.Now().Add(time.Hour), time.Second))

			claims, err = newStandardClaims("ginkgo", ProvisioningPurpose, 5*time.Hour, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Issuer).To(Equal("ginkgo"))
			Expect(claims.Purpose).To(Equal(ProvisioningPurpose))
			Expect(claims.Subject).To(Equal(string(ProvisioningPurpose)))
			Expect(claims.IssuedAt.Time).To(BeTemporally("~", time.Now(), time.Second))
			Expect(claims.ExpiresAt.Time).To(BeTemporally("~", time.Now().Add(5*time.Hour), time.Second))
		})
	})
})
