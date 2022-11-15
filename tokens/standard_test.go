// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"time"

	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StandardClaims", func() {
	var (
		c    StandardClaims
		priK ed25519.PrivateKey
		pubK ed25519.PublicKey
		err  error
	)

	BeforeEach(func() {
		c = StandardClaims{}
		pubK, priK, err = iu.Ed25519KeyPair()
		Expect(err).ToNot(HaveOccurred())

	})

	Describe("Chain Issuer", func() {
		BeforeEach(func() {
			c = StandardClaims{}
			c.ID = iu.UniqueID()
		})

		Describe("verifyIssuerExpiry", func() {
			BeforeEach(func() {
				c = StandardClaims{}
				c.Issuer = "C-x.x"
				c.TrustChainSignature = "stub.sig"
				c.IssuerExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
			})

			It("Should only verify on tokens issues by a chain issues", func() {
				c.Issuer = "I-x.x"
				Expect(c.verifyIssuerExpiry(false)).To(BeTrue())
				Expect(c.verifyIssuerExpiry(true)).To(BeFalse())
			})

			It("Should detect missing tcs", func() {
				c.TrustChainSignature = ""
				Expect(c.verifyIssuerExpiry(true)).To(BeFalse())
				Expect(c.verifyIssuerExpiry(false)).To(BeTrue())
			})

			It("Should detect missing issuer expiry", func() {
				c.IssuerExpiresAt = nil
				Expect(c.verifyIssuerExpiry(true)).To(BeFalse())
				Expect(c.verifyIssuerExpiry(false)).To(BeTrue())
			})

			It("Should correctly detect expiry", func() {
				Expect(c.verifyIssuerExpiry(true)).To(BeTrue())
				Expect(c.verifyIssuerExpiry(false)).To(BeTrue())

				c.IssuerExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Minute))
				Expect(c.verifyIssuerExpiry(true)).To(BeFalse())
				Expect(c.verifyIssuerExpiry(false)).To(BeFalse())
			})
		})

		Describe("SetChainIssuer", func() {
			It("Should require basic data", func() {
				ci := &ClientIDClaims{}
				Expect(c.SetChainIssuer(ci)).To(MatchError("id not set"))

				ci.ID = "x"
				Expect(c.SetChainIssuer(ci)).To(MatchError("public key not set"))
			})

			It("Should set the correct issuer", func() {
				ci := &ClientIDClaims{}
				ci.ID = iu.UniqueID()
				ci.PublicKey = hex.EncodeToString(pubK[:])
				Expect(c.SetChainIssuer(ci)).To(Succeed())
				Expect(c.Issuer).To(Equal(fmt.Sprintf("C-%s.%s", ci.ID, ci.PublicKey)))
			})
		})

		Describe("ChainIssuerData", func() {
			It("Should require minimal data", func() {
				c.ID = ""
				_, err = c.ChainIssuerData("x")
				Expect(err).To(MatchError("id not set"))

				c.ID = iu.UniqueID()
				_, err = c.ChainIssuerData("x")
				Expect(err).To(MatchError("issuer not set"))

				c.Issuer = "X-Issuer"
				_, err = c.ChainIssuerData("x")
				Expect(err).To(MatchError("invalid issuer prefix"))

				c.Issuer = "C-x"
				_, err = c.ChainIssuerData("x")
				Expect(err).To(MatchError("invalid issuer data"))
			})

			It("Should issue the correct data", func() {
				ci := &ClientIDClaims{}
				ci.ID = iu.UniqueID()
				ci.PublicKey = hex.EncodeToString(pubK[:])
				Expect(c.SetChainIssuer(ci)).To(Succeed())

				dat, err := c.ChainIssuerData("x")
				Expect(err).ToNot(HaveOccurred())
				Expect(dat).To(Equal([]byte(fmt.Sprintf("%s.x", c.ID))))
			})
		})

		Describe("IsSignedByIssuer", func() {
			It("Should detect badly formed issuers", func() {
				c.Issuer = "C-x"
				c.PublicKey = hex.EncodeToString(pubK)
				c.TrustChainSignature = "x"
				c.ID = "ID"

				ok, _, err := c.IsSignedByIssuer(pubK)
				Expect(err).To(MatchError("invalid issuer content"))
				Expect(ok).To(BeFalse())

				c.Issuer = "C-.x"
				ok, _, err = c.IsSignedByIssuer(pubK)
				Expect(err).To(MatchError("invalid id in issuer"))
				Expect(ok).To(BeFalse())

				c.Issuer = "C-y."
				ok, _, err = c.IsSignedByIssuer(pubK)
				Expect(err).To(MatchError("invalid public key in issuer"))
				Expect(ok).To(BeFalse())

				c.Issuer = "C-!.y"
				ok, _, err = c.IsSignedByIssuer(pubK)
				Expect(err).To(MatchError("invalid public key in issuer data"))
				Expect(ok).To(BeFalse())
			})

			It("Should detect badly formed trust chain sigs", func() {
				c.Issuer = "C-x." + hex.EncodeToString(pubK)
				c.PublicKey = hex.EncodeToString(pubK)
				c.ID = iu.UniqueID()
				c.TrustChainSignature = "X"

				ok, _, err := c.IsSignedByIssuer(pubK)
				Expect(err).To(MatchError("invalid trust chain signature"))
				Expect(ok).To(BeFalse())

				c.TrustChainSignature = "."
				ok, _, err = c.IsSignedByIssuer(pubK)
				Expect(err).To(MatchError("invalid trust chain signature"))
				Expect(ok).To(BeFalse())

				c.TrustChainSignature = "foo.!!"
				ok, _, err = c.IsSignedByIssuer(pubK)
				Expect(err).To(MatchError("invalid signature in chain signature: encoding/hex: invalid byte: U+0021 '!'"))
				Expect(ok).To(BeFalse())
			})

			It("Should detect incorrect signatures", func() {
				// the org issuer
				issuePubK, issuerPriK, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				handlerPubK, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				userPubK, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				// the handler signed by the org issuer
				handler, err := NewClientIDClaims("choria=handler", nil, "", nil, "", "", time.Minute, nil, handlerPubK)
				Expect(err).ToNot(HaveOccurred())
				handler.SetOrgIssuer(issuePubK)
				hdat, err := handler.OrgIssuerChainData()
				Expect(err).ToNot(HaveOccurred())
				hsig, err := iu.Ed25519Sign(issuerPriK, hdat)
				Expect(err).ToNot(HaveOccurred())
				handler.TrustChainSignature = "invalid"

				// it looks good so it passes but with verify fails
				Expect(handler.IsChainedIssuer(false)).To(BeTrue())
				Expect(handler.IsChainedIssuer(true)).To(BeFalse())

				handler.TrustChainSignature = hex.EncodeToString(hsig)

				// now it looks good and it is good
				Expect(handler.IsChainedIssuer(false)).To(BeTrue())
				Expect(handler.IsChainedIssuer(true)).To(BeTrue())

				// a user issued by the handler
				user, err := NewClientIDClaims("choria=user", nil, "", nil, "", "", time.Minute, nil, userPubK)
				Expect(err).ToNot(HaveOccurred())
				Expect(user.SetChainIssuer(handler)).To(Succeed())
				user.SetChainUserTrustSignature(handler, []byte("invalid sig"))
				ok, _, err := user.IsSignedByIssuer(issuePubK)
				Expect(err).To(MatchError("invalid chain signature"))
				Expect(ok).To(BeFalse())
			})

			It("Should detect correct signatures", func() {
				// the org issuer
				issuePubK, issuerPriK, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				handlerPubK, handlerPrik, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				userPubK, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				// the handler signed by the org issuer
				handler, err := NewClientIDClaims("choria=handler", nil, "", nil, "", "", time.Minute, nil, handlerPubK)
				Expect(err).ToNot(HaveOccurred())
				handler.SetOrgIssuer(issuePubK)
				hdat, err := handler.OrgIssuerChainData()
				Expect(err).ToNot(HaveOccurred())
				hsig, err := iu.Ed25519Sign(issuerPriK, hdat)
				Expect(err).ToNot(HaveOccurred())
				handler.TrustChainSignature = hex.EncodeToString(hsig)

				// a user issued by the handler
				user, err := NewClientIDClaims("choria=user", nil, "", nil, "", "", time.Minute, nil, userPubK)
				Expect(err).ToNot(HaveOccurred())
				Expect(user.SetChainIssuer(handler)).To(Succeed())
				udat, err := user.ChainIssuerData(handler.TrustChainSignature)
				Expect(err).ToNot(HaveOccurred())
				usig, err := iu.Ed25519Sign(handlerPrik, udat)
				Expect(err).ToNot(HaveOccurred())
				user.SetChainUserTrustSignature(handler, usig)
				ok, _, err := user.IsSignedByIssuer(issuePubK)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
	})

	Describe("Organization Issuer", func() {
		Describe("IsSignedByIssuer", func() {
			It("Should expect minimally correct data", func() {
				check := func(pk ed25519.PublicKey, expect error) {
					ok, _, err := c.IsSignedByIssuer(pk)
					if expect == nil && !ok {
						Fail(fmt.Sprintf("Expected to be ok but got %v", err))
					}

					Expect(err).To(MatchError(expect))
					Expect(ok).To(BeFalse())
				}

				check(nil, fmt.Errorf("no issuer set"))
				c.Issuer = "issuer"

				check(nil, fmt.Errorf("no public key set"))
				c.PublicKey = hex.EncodeToString(pubK)

				check(pubK, fmt.Errorf("no trust chain signature set"))
				c.TrustChainSignature = "x"

				check(pubK, fmt.Errorf("id not set"))
			})

			It("Should detect invalid issuers", func() {
				c.Issuer = "issuer"
				c.PublicKey = hex.EncodeToString(pubK)
				c.TrustChainSignature = "x"
				c.ID = "ID"

				ok, _, err := c.IsSignedByIssuer(pubK)
				Expect(err).To(MatchError("unsupported issuer format"))
				Expect(ok).To(BeFalse())
			})

			It("Should detect invalid signatures", func() {
				c.PublicKey = hex.EncodeToString(pubK)
				c.Issuer = fmt.Sprintf("I-%s", c.PublicKey)
				c.TrustChainSignature = "X"
				c.ID = "ID"
				ok, _, err := c.IsSignedByIssuer(pubK)
				Expect(err).To(MatchError("invalid trust chain signature: encoding/hex: invalid byte: U+0058 'X'"))
				Expect(ok).To(BeFalse())
			})

			It("Should detect wrong signatures", func() {
				c.PublicKey = hex.EncodeToString(pubK)
				c.Issuer = fmt.Sprintf("I-%s", c.PublicKey)
				c.ID = iu.UniqueID()

				sig, err := iu.Ed25519Sign(priK, []byte("wrong"))
				Expect(err).ToNot(HaveOccurred())
				c.TrustChainSignature = hex.EncodeToString(sig)

				ok, _, err := c.IsSignedByIssuer(pubK)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})

			It("Should detect correct signatures", func() {
				c.PublicKey = hex.EncodeToString(pubK)
				c.Issuer = fmt.Sprintf("I-%s", c.PublicKey)
				c.ID = iu.UniqueID()

				dat, err := c.OrgIssuerChainData()
				Expect(err).ToNot(HaveOccurred())

				sig, err := iu.Ed25519Sign(priK, dat)
				Expect(err).ToNot(HaveOccurred())
				c.TrustChainSignature = hex.EncodeToString(sig)

				ok, pk, err := c.IsSignedByIssuer(pubK)
				Expect(err).ToNot(HaveOccurred())
				Expect(pk).To(Equal(pubK))
				Expect(ok).To(BeTrue())
			})
		})

		Describe("SetOrgIssuer", func() {
			It("Should set the correct issuer", func() {
				pubK, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				c.SetOrgIssuer(pubK)
				Expect(c.Issuer).To(Equal(fmt.Sprintf("I-%x", pubK)))
			})
		})

		Describe("OrgIssuerChainData", func() {
			It("Should fail for no ID", func() {
				d, err := c.OrgIssuerChainData()
				Expect(d).To(HaveLen(0))
				Expect(err).To(MatchError("no token id set"))
			})

			It("Should fail for no PublicKey", func() {
				c.ID = "x"
				d, err := c.OrgIssuerChainData()
				Expect(d).To(HaveLen(0))
				Expect(err).To(MatchError("no public key set"))
			})

			It("Should create the correct data", func() {
				c.ID = "id"
				c.PublicKey = "pubk"
				d, err := c.OrgIssuerChainData()
				Expect(err).ToNot(HaveOccurred())
				Expect(d).To(Equal([]byte("id.pubk")))
			})
		})
	})
})
