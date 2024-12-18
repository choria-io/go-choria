// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"crypto/ed25519"
	"crypto/tls"
	"encoding/hex"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/tokens"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"go.uber.org/mock/gomock"
)

func TestFileSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Security/Choria")
}

var _ = Describe("Providers/Security/Choria", func() {
	var (
		err     error
		cfg     *Config
		prov    *ChoriaSecurity
		fw      inter.Framework
		mockCtl *gomock.Controller
		logbuf  *gbytes.Buffer
	)

	BeforeEach(func() {
		cfg = &Config{}
		mockCtl = gomock.NewController(GinkgoT())
		logbuf = gbytes.NewBuffer()
		fw, _ = imock.NewFrameworkForTests(mockCtl, logbuf)

		prov, err = New(WithConfig(cfg), WithLog(fw.Logger("ginkgo")))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		mockCtl.Finish()
	})

	It("Should be a valid provider", func() {
		Expect(any(prov).(inter.SecurityProvider).Provider()).To(Equal("choria"))
		Expect(prov.BackingTechnology()).To(Equal(inter.SecurityTechnologyED25519JWT))
		Expect(prov.Provider()).To(Equal("choria"))
	})

	Describe("WithChoriaConfig", func() {
		It("Should verify trusted signer key lengths", func() {
			cc := fw.Configuration()
			cc.Choria.ChoriaSecurityTrustedSigners = []string{hex.EncodeToString([]byte("x"))}
			err = WithChoriaConfig(cc)(prov)
			Expect(err).To(MatchError("invalid ed25519 public key size: 78: 1"))
		})

		It("Should support disabling reply signatures", func() {
			cc := fw.Configuration()
			Expect(WithChoriaConfig(cc)(prov)).To(Succeed())
			Expect(prov.conf.SignedReplies).To(BeTrue())

			cc.Choria.ChoriaSecuritySignReplies = false
			Expect(WithChoriaConfig(cc)(prov)).To(Succeed())
			Expect(prov.conf.SignedReplies).To(BeFalse())
		})

		It("Should load all signers", func() {
			cc := fw.Configuration()

			pk1, _, err := iu.Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())
			cc.Choria.ChoriaSecurityTrustedSigners = append(cc.Choria.ChoriaSecurityTrustedSigners, hex.EncodeToString(pk1))
			pk2, _, err := iu.Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())
			cc.Choria.ChoriaSecurityTrustedSigners = append(cc.Choria.ChoriaSecurityTrustedSigners, hex.EncodeToString(pk2))

			Expect(WithChoriaConfig(cc)(prov)).To(Succeed())
			Expect(prov.conf.TrustedTokenSigners).To(HaveLen(2))
			Expect(prov.conf.TrustedTokenSigners[0]).To(Equal(pk1))
			Expect(prov.conf.TrustedTokenSigners[1]).To(Equal(pk2))
			Expect(prov.conf.SignedReplies).To(BeTrue())
		})
	})

	Describe("Validate", func() {
		BeforeEach(func() {
			cfg.Identity = "ginkgo"
			cfg.TokenFile = "/nonexisting"
			cfg.SeedFile = "/nonexisting"
			cfg.TrustedTokenSigners = []ed25519.PublicKey{[]byte("x")}
		})

		It("Should ensure a valid configuration exist", func() {
			prov.conf = nil
			errs, ok := prov.Validate()
			Expect(ok).To(BeFalse())
			Expect(errs).To(Equal([]string{"configuration not given"}))
		})

		It("Should ensure a logger is given", func() {
			prov.log = nil
			errs, ok := prov.Validate()
			Expect(ok).To(BeFalse())
			Expect(errs).To(Equal([]string{"logger not given"}))
		})

		It("Should ensure a identity is set", func() {
			cfg.Identity = ""
			errs, ok := prov.Validate()
			Expect(ok).To(BeFalse())
			Expect(errs).To(Equal([]string{"identity could not be determine automatically via Choria or was not supplied"}))
		})

		It("Should ensure a JWT token path is set", func() {
			cfg.TokenFile = ""
			errs, ok := prov.Validate()
			Expect(ok).To(BeFalse())
			Expect(errs).To(Equal([]string{"the path to the JWT token is not configured"}))
		})

		It("Should ensure a ed25519 seed path is set", func() {
			cfg.SeedFile = ""
			errs, ok := prov.Validate()
			Expect(ok).To(BeFalse())
			Expect(errs).To(Equal([]string{"the path to the ed25519 seed is not configured"}))
		})

		It("Should ensure trusted token signers are set", func() {
			cfg.TrustedTokenSigners = []ed25519.PublicKey{}
			errs, ok := prov.Validate()
			Expect(errs).To(BeNil())
			Expect(ok).To(BeTrue())

			cfg.InitiatedByServer = true
			errs, ok = prov.Validate()
			Expect(errs).To(Equal([]string{"no trusted token signers or issuers configured"}))
			Expect(ok).To(BeFalse())
		})

		It("Should prevent issuers and trusted signers from being used concurrently", func() {
			cfg.Issuers = map[string]ed25519.PublicKey{"choria": cfg.TrustedTokenSigners[0]}
			cfg.InitiatedByServer = true
			errs, ok := prov.Validate()
			Expect(errs).To(Equal([]string{"can only configure one of trusted token signers or issuers"}))
			Expect(ok).To(BeFalse())
		})
	})

	Describe("SignBytes", func() {
		It("Should sign using the seed file", func() {
			tf, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tf.Name())
			tf.Close()

			prov.conf.SeedFile = tf.Name()

			_, prik, err := iu.Ed25519KeyPairToFile(tf.Name())
			Expect(err).ToNot(HaveOccurred())

			sig, err := prov.SignBytes([]byte("too many secrets"))
			Expect(err).ToNot(HaveOccurred())

			esig := ed25519.Sign(prik, []byte("too many secrets"))

			Expect(sig).To(Equal(esig))
		})
	})

	Describe("VerifySignatureBytes", func() {
		Describe("Issuer Based", func() {
			var (
				// the issuer seed
				issuerSeedFile string
				issuerPubk     ed25519.PublicKey
				issuerPrik     ed25519.PrivateKey

				// aaa login service can make new clients
				aaaSvcSeedFile, aaaSvcJWTFile string
				aaaServicePubk                ed25519.PublicKey
				aaaServicePrik                ed25519.PrivateKey

				// aaa signer service
				aaaSignerSeedFile, aaaSignerJWTFile string
				aaaSignerPubk                       ed25519.PublicKey
				aaaSignerPrik                       ed25519.PrivateKey

				// caller that needs a signer to authorize it
				delegatedCallerSeedFile, delegatedCallerJWTFile string

				// correctly signed but without fleet access
				nonFleetSeedFile, nonFleetJWTFile string
				nonFleetPubk                      ed25519.PublicKey
				nonFleetPrik                      ed25519.PrivateKey

				// a correctly signed client with fleet access
				callerSeedFile, callerJWTFile string
				callerPubk                    ed25519.PublicKey
				callerPrik                    ed25519.PrivateKey

				// server signed by the issuer
				serverSeedFile, serverJWTFile string
				serverPubk                    ed25519.PublicKey
				serverPrik                    ed25519.PrivateKey

				// provisioner.jwt
				provJWTFile string
			)

			BeforeEach(func() {
				td, err := os.MkdirTemp("", "")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() { os.RemoveAll(td) })

				issuerSeedFile = filepath.Join(td, "issuer.seed")
				aaaSvcSeedFile = filepath.Join(td, "aaa-login.seed")
				aaaSvcJWTFile = filepath.Join(td, "aaa-login.jwt")
				aaaSignerSeedFile = filepath.Join(td, "aaa-signer.seed")
				aaaSignerJWTFile = filepath.Join(td, "aaa-signer.jwt")
				delegatedCallerSeedFile = filepath.Join(td, "delegated.seed")
				delegatedCallerJWTFile = filepath.Join(td, "delegated.jwt")
				provJWTFile = filepath.Join(td, "provisioner.jwt")
				nonFleetSeedFile = filepath.Join(td, "nonfleet.seed")
				nonFleetJWTFile = filepath.Join(td, "nonfleet.jwt")
				callerSeedFile = filepath.Join(td, "caller.seed")
				callerJWTFile = filepath.Join(td, "caller.jwt")
				serverSeedFile = filepath.Join(td, "server.seed")
				serverJWTFile = filepath.Join(td, "server.jwt")

				// issuer mode org issuer
				issuerPubk, issuerPrik, err = iu.Ed25519KeyPairToFile(issuerSeedFile)
				Expect(err).ToNot(HaveOccurred())

				// provisioner.jwt signed by the issuer directly
				provToken, err := tokens.NewProvisioningClaims(true, true, "x", "x", "x", nil, "example.net", "", "", "ginkgo", "", time.Hour)
				Expect(err).ToNot(HaveOccurred())
				Expect(provToken.Purpose).To(Equal(tokens.ProvisioningPurpose))
				Expect(tokens.SaveAndSignTokenWithKeyFile(provToken, issuerSeedFile, provJWTFile, 0600)).To(Succeed())

				// chain issuer for aaa service
				aaaServicePubk, aaaServicePrik, err = iu.Ed25519KeyPairToFile(aaaSvcSeedFile)
				Expect(err).ToNot(HaveOccurred())
				aaaToken, err := tokens.NewClientIDClaims("aaasvc-login", nil, "choria", nil, "", "Ginkgo Tests", time.Hour, nil, aaaServicePubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(aaaToken.AddOrgIssuerData(issuerPrik)).To(Succeed())
				Expect(tokens.SaveAndSignTokenWithKeyFile(aaaToken, issuerSeedFile, aaaSvcJWTFile, 0600)).To(Succeed())

				// aaa signer
				aaaSignerPubk, aaaSignerPrik, err = iu.Ed25519KeyPairToFile(aaaSignerSeedFile)
				Expect(err).ToNot(HaveOccurred())
				aaaSignerToken, err := tokens.NewClientIDClaims("aaasvc-signer", nil, "choria", nil, "", "Ginkgo Tests", time.Hour, &tokens.ClientPermissions{AuthenticationDelegator: true}, aaaSignerPubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(aaaSignerToken.Permissions.AuthenticationDelegator).To(BeTrue())
				Expect(aaaSignerToken.AddOrgIssuerData(issuerPrik)).To(Succeed())
				Expect(tokens.SaveAndSignTokenWithKeyFile(aaaSignerToken, issuerSeedFile, aaaSignerJWTFile, 0600)).To(Succeed())

				// clients thats correctly signed etc, but cant access the fleet
				nonFleetPubk, nonFleetPrik, err = iu.Ed25519KeyPairToFile(nonFleetSeedFile)
				Expect(err).ToNot(HaveOccurred())
				nonFleetToken, err := tokens.NewClientIDClaims("delegated_caller", nil, "choria", nil, "", "Ginkgo Tests", time.Hour, nil, nonFleetPubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(nonFleetToken.AddOrgIssuerData(issuerPrik)).To(Succeed())
				Expect(tokens.SaveAndSignTokenWithKeyFile(nonFleetToken, issuerSeedFile, nonFleetJWTFile, 0600)).To(Succeed())

				// a valid non delegated caller with fleet access
				callerPubk, callerPrik, err = iu.Ed25519KeyPairToFile(callerSeedFile)
				Expect(err).ToNot(HaveOccurred())
				callerToken, err := tokens.NewClientIDClaims("caller", nil, "choria", nil, "", "Ginkgo Tests", time.Hour, &tokens.ClientPermissions{FleetManagement: true}, callerPubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(callerToken.AddOrgIssuerData(issuerPrik)).To(Succeed())
				Expect(tokens.SaveAndSignTokenWithKeyFile(callerToken, issuerSeedFile, callerJWTFile, 0600)).To(Succeed())

				// client that requires delegated signer
				delegatedCallerPubk, _, err := iu.Ed25519KeyPairToFile(delegatedCallerSeedFile)
				Expect(err).ToNot(HaveOccurred())
				delegatedCallerToken, err := tokens.NewClientIDClaims("delegated_caller", nil, "choria", nil, "", "Ginkgo Tests", time.Hour, &tokens.ClientPermissions{SignedFleetManagement: true}, delegatedCallerPubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(delegatedCallerToken.Permissions.SignedFleetManagement).To(BeTrue())
				Expect(delegatedCallerToken.AddChainIssuerData(aaaToken, aaaServicePrik)).To(Succeed())
				Expect(tokens.SaveAndSignTokenWithKeyFile(delegatedCallerToken, aaaSvcSeedFile, delegatedCallerJWTFile, 0600)).To(Succeed())

				// server token
				serverPubk, serverPrik, err = iu.Ed25519KeyPairToFile(serverSeedFile)
				Expect(err).ToNot(HaveOccurred())
				serverToken, err := tokens.NewServerClaims("example.net", []string{"choria"}, "choria", nil, nil, serverPubk, "ginkgo", time.Hour)
				Expect(err).ToNot(HaveOccurred())
				Expect(serverToken.Purpose).To(Equal(tokens.ServerPurpose))
				Expect(serverToken.AddChainIssuerData(aaaToken, aaaServicePrik)).To(Succeed())
				Expect(tokens.SaveAndSignTokenWithKeyFile(serverToken, aaaSvcSeedFile, serverJWTFile, 0600)).To(Succeed())

				cfg.TrustedTokenSigners = []ed25519.PublicKey{}
				cfg.Issuers = map[string]ed25519.PublicKey{
					"choria": issuerPubk,
				}
			})

			It("Should fail for no public parts", func() {
				should, signer := prov.VerifySignatureBytes(nil, nil)
				Expect(should).To(BeFalse())
				Expect(signer).To(Equal(""))
				Expect(logbuf).To(gbytes.Say("Received a signature verification request with no public parts"))
			})

			Describe("Caller signatures", func() {
				It("Should detect invalid token purposes", func() {
					pub, err := os.ReadFile(provJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("Cannot verify byte signatures using a choria_provisioning token"))
				})

				It("Should deny client tokens that require a delegator to sign requests", func() {
					pub, err := os.ReadFile(delegatedCallerJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("access denied: requires authority delegation"))
				})

				It("Should deny client tokens that does not have fleet management access", func() {
					pub, err := os.ReadFile(nonFleetJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("access denied: does not have fleet management access"))
				})

				It("Should support client tokens", func() {
					sig, err := iu.Ed25519Sign(callerPrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					pub, err := os.ReadFile(callerJWTFile)
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, pub)
					Expect(should).To(BeTrue())
					Expect(signer).To(Equal("caller"))
				})

				It("Should support server tokens", func() {
					sig, err := iu.Ed25519Sign(serverPrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					pub, err := os.ReadFile(serverJWTFile)
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, pub)
					Expect(should).To(BeTrue())
					Expect(signer).To(Equal("example.net"))
				})
			})

			Describe("Delegated signatures", func() {
				It("Should detect invalid token purposes", func() {
					pub, err := os.ReadFile(provJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("Cannot verify byte signatures using a choria_provisioning token"))
				})

				It("Should ensure that the signer has delegated signature permissions", func() {
					pub, err := os.ReadFile(callerJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("token attempted to sign a request as delegator without required delegator permission"))
				})

				It("Should ensure the caller has fleet access", func() {
					nonFleet, err := os.ReadFile(nonFleetJWTFile)
					Expect(err).ToNot(HaveOccurred())

					signerToken, err := os.ReadFile(aaaSignerJWTFile)
					Expect(err).ToNot(HaveOccurred())

					sig, err := iu.Ed25519Sign(nonFleetPrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, nonFleet, signerToken)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("caller token cannot be used without fleet management access"))
				})

				It("Should fail for server tokens", func() {
					signerToken, err := os.ReadFile(aaaSignerJWTFile)
					Expect(err).ToNot(HaveOccurred())

					server, err := os.ReadFile(serverJWTFile)
					Expect(err).ToNot(HaveOccurred())

					sig, err := iu.Ed25519Sign(aaaSignerPrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, server, signerToken)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("could not load caller token using the same signer as the delegator: not a client token"))
				})

				It("Should support client tokens", func() {
					signerToken, err := os.ReadFile(aaaSignerJWTFile)
					Expect(err).ToNot(HaveOccurred())

					caller, err := os.ReadFile(callerJWTFile)
					Expect(err).ToNot(HaveOccurred())

					sig, err := iu.Ed25519Sign(aaaSignerPrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, caller, signerToken)
					Expect(should).To(BeTrue())
					Expect(signer).To(Equal("aaasvc-signer"))
				})
			})
		})

		Describe("Trusted Signers Based", func() {
			var (
				// a delegator that signs other users requests, has no fleet management feature access
				delegateSeedFile, delegateJWTFile string

				// a caller that may only access the fleet via delegator signature
				delegatedCallerSeedFile, delegatedCallerJWTFile string

				// a caller that can make fleet requests without signature
				callerSeedFile, callerJWTFile string

				// a provisioner token
				provJWTFile string

				// a server purpose token
				serverSeedFile, serverJWTFile string

				// signs all the jwt files
				signerSeedFile string

				delegatePrik, callerPrik, serverPrik ed25519.PrivateKey
				delegatePubk, callerPubk, serverPubk ed25519.PublicKey
			)

			BeforeEach(func() {
				td, err := os.MkdirTemp("", "")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() { os.RemoveAll(td) })

				signerSeedFile = filepath.Join(td, "signer.seed")
				delegateSeedFile = filepath.Join(td, "delegate.seed")
				delegateJWTFile = filepath.Join(td, "delegate.jwt")
				delegatedCallerSeedFile = filepath.Join(td, "delegated_caller.seed")
				delegatedCallerJWTFile = filepath.Join(td, "delegated_caller.jwt")
				callerSeedFile = filepath.Join(td, "caller.seed")
				callerJWTFile = filepath.Join(td, "caller.jwt")
				serverSeedFile = filepath.Join(td, "server.seed")
				serverJWTFile = filepath.Join(td, "server.jwt")
				provJWTFile = filepath.Join(td, "provisioner.jwt")

				// signs all the jwt tokens
				signerPubk, _, err := iu.Ed25519KeyPairToFile(signerSeedFile)
				Expect(err).ToNot(HaveOccurred())

				// some other seed just to test multi trusted signer feature
				otherPubk, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				// just shuffle the trusted tokens to test multi signer support automatically
				if rand.N(10) <= 5 {
					cfg.TrustedTokenSigners = []ed25519.PublicKey{otherPubk, signerPubk}
				} else {
					cfg.TrustedTokenSigners = []ed25519.PublicKey{signerPubk, otherPubk}
				}

				// signs delegated requests
				delegatePubk, delegatePrik, err = iu.Ed25519KeyPairToFile(delegateSeedFile)
				Expect(err).ToNot(HaveOccurred())
				delegateToken, err := tokens.NewClientIDClaims("delegate", nil, "choria", nil, "", "Ginkgo Tests", time.Hour, &tokens.ClientPermissions{AuthenticationDelegator: true}, delegatePubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(delegateToken.Permissions.AuthenticationDelegator).To(BeTrue())
				Expect(tokens.SaveAndSignTokenWithKeyFile(delegateToken, signerSeedFile, delegateJWTFile, 0600)).To(Succeed())

				// a caller that needs delegation
				delegatedCallerPubk, _, err := iu.Ed25519KeyPairToFile(delegatedCallerSeedFile)
				Expect(err).ToNot(HaveOccurred())
				delegatedCallerToken, err := tokens.NewClientIDClaims("delegated_caller", nil, "choria", nil, "", "Ginkgo Tests", time.Hour, &tokens.ClientPermissions{SignedFleetManagement: true}, delegatedCallerPubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(delegatedCallerToken.Permissions.SignedFleetManagement).To(BeTrue())
				Expect(tokens.SaveAndSignTokenWithKeyFile(delegatedCallerToken, signerSeedFile, delegatedCallerJWTFile, 0600)).To(Succeed())

				// caller that can sign itself
				callerPubk, callerPrik, err = iu.Ed25519KeyPairToFile(callerSeedFile)
				Expect(err).ToNot(HaveOccurred())
				callerToken, err := tokens.NewClientIDClaims("caller", nil, "choria", nil, "", "Ginkgo Tests", time.Hour, &tokens.ClientPermissions{FleetManagement: true}, callerPubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(callerToken.Permissions.FleetManagement).To(BeTrue())
				Expect(tokens.SaveAndSignTokenWithKeyFile(callerToken, signerSeedFile, callerJWTFile, 0600)).To(Succeed())

				// server token
				serverPubk, serverPrik, err = iu.Ed25519KeyPairToFile(serverSeedFile)
				Expect(err).ToNot(HaveOccurred())
				serverToken, err := tokens.NewServerClaims("example.net", []string{"choria"}, "choria", nil, nil, serverPubk, "ginkgo", time.Hour)
				Expect(err).ToNot(HaveOccurred())
				Expect(serverToken.Purpose).To(Equal(tokens.ServerPurpose))
				Expect(tokens.SaveAndSignTokenWithKeyFile(serverToken, signerSeedFile, serverJWTFile, 0600)).To(Succeed())

				// a provisioner purpose token
				provToken, err := tokens.NewProvisioningClaims(true, true, "x", "x", "x", nil, "example.net", "", "", "ginkgo", "", time.Hour)
				Expect(err).ToNot(HaveOccurred())
				Expect(provToken.Purpose).To(Equal(tokens.ProvisioningPurpose))
				Expect(tokens.SaveAndSignTokenWithKeyFile(provToken, signerSeedFile, provJWTFile, 0600)).To(Succeed())
			})

			It("Should fail for no public parts", func() {
				should, signer := prov.VerifySignatureBytes(nil, nil)
				Expect(should).To(BeFalse())
				Expect(signer).To(Equal(""))
				Expect(logbuf).To(gbytes.Say("Received a signature verification request with no public parts"))
			})

			Describe("Caller signatures", func() {
				It("Should detect invalid token purposes", func() {
					pub, err := os.ReadFile(provJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("Cannot verify byte signatures using a choria_provisioning token"))
				})

				It("Should deny client tokens that require a delegator to sign requests", func() {
					pub, err := os.ReadFile(delegatedCallerJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("Could not verify signature by caller: access denied: requires authority delegation"))
				})

				It("Should deny client tokens that does not have fleet management access", func() {
					pub, err := os.ReadFile(delegateJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("Could not verify signature by caller: access denied: does not have fleet management access"))
				})

				It("Should support client tokens", func() {
					sig, err := iu.Ed25519Sign(callerPrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					pub, err := os.ReadFile(callerJWTFile)
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, pub)
					Expect(should).To(BeTrue())
					Expect(signer).To(Equal("caller"))
				})

				It("Should support server tokens", func() {
					sig, err := iu.Ed25519Sign(serverPrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					pub, err := os.ReadFile(serverJWTFile)
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, pub)
					Expect(should).To(BeTrue())
					Expect(signer).To(Equal("example.net"))
				})
			})

			Describe("Delegated signatures", func() {
				It("Should detect invalid token purposes", func() {
					pub, err := os.ReadFile(provJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("Cannot verify byte signatures using a choria_provisioning token"))
				})

				It("Should ensure that the signer has delegated signature permissions", func() {
					pub, err := os.ReadFile(callerJWTFile)
					Expect(err).ToNot(HaveOccurred())
					should, signer := prov.VerifySignatureBytes(nil, nil, nil, pub)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("token attempted to sign a request as delegator without required delegator permission"))
				})

				It("Should ensure the caller has fleet access", func() {
					delegate, err := os.ReadFile(delegateJWTFile)
					Expect(err).ToNot(HaveOccurred())

					sig, err := iu.Ed25519Sign(delegatePrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, delegate, delegate)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("caller token cannot be used without fleet management access"))
				})

				It("Should fail for server tokens", func() {
					delegate, err := os.ReadFile(delegateJWTFile)
					Expect(err).ToNot(HaveOccurred())

					server, err := os.ReadFile(serverJWTFile)
					Expect(err).ToNot(HaveOccurred())

					sig, err := iu.Ed25519Sign(delegatePrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, server, delegate)
					Expect(should).To(BeFalse())
					Expect(signer).To(Equal(""))
					Expect(logbuf).To(gbytes.Say("could not load caller token using the same signer as the delegator: not a client token"))
				})

				It("Should support client tokens", func() {
					delegate, err := os.ReadFile(delegateJWTFile)
					Expect(err).ToNot(HaveOccurred())

					caller, err := os.ReadFile(callerJWTFile)
					Expect(err).ToNot(HaveOccurred())

					sig, err := iu.Ed25519Sign(delegatePrik, []byte("too many secrets"))
					Expect(err).ToNot(HaveOccurred())

					should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, caller, delegate)
					Expect(should).To(BeTrue())
					Expect(signer).To(Equal("delegate"))
				})
			})
		})
	})

	Describe("PublicCert", func() {
		It("Should load the correct public cert", func() {
			pc, err := prov.PublicCert()
			Expect(err).To(HaveOccurred())
			Expect(pc).To(BeNil())

			prov.conf.Key = filepath.Join("..", "testdata", "good", "private_keys", "rip.mcollective.pem")
			prov.conf.Certificate = filepath.Join("..", "testdata", "good", "certs", "rip.mcollective.pem")

			pc, err = prov.PublicCert()
			Expect(err).ToNot(HaveOccurred())

			Expect(pc.Subject.String()).To(Equal("CN=rip.mcollective"))
		})
	})

	Describe("TLSConfig", func() {
		It("Should support operating without cert/key/ca", func() {
			c, err := prov.TLSConfig()
			Expect(err).ToNot(HaveOccurred())

			Expect(c.InsecureSkipVerify).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			Expect(c.Certificates).To(BeEmpty())
		})

		It("Should produce a valid TLS Config", func() {
			prov.conf.Key = filepath.Join("..", "testdata", "good", "private_keys", "rip.mcollective.pem")
			prov.conf.Certificate = filepath.Join("..", "testdata", "good", "certs", "rip.mcollective.pem")
			prov.conf.CA = filepath.Join("..", "testdata", "good", "certs", "ca.pem")

			c, err := prov.TLSConfig()
			Expect(err).ToNot(HaveOccurred())

			Expect(c.InsecureSkipVerify).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())

			cert, err := tls.LoadX509KeyPair(prov.conf.Certificate, prov.conf.Key)
			Expect(err).ToNot(HaveOccurred())

			Expect(c.Certificates).To(HaveLen(1))
			Expect(c.Certificates[0].Certificate).To(Equal(cert.Certificate))
		})

		It("Should support disabling tls verify", func() {
			cfg.DisableTLSVerify = true

			c, err := prov.TLSConfig()
			Expect(err).ToNot(HaveOccurred())

			Expect(c.InsecureSkipVerify).To(BeTrue())
		})
	})

})
