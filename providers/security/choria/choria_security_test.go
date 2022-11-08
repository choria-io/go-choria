// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"crypto/ed25519"
	"crypto/tls"
	"encoding/hex"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tokens"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
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
		rand.Seed(time.Now().UnixNano())

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
			Expect(errs).To(Equal([]string{"no trusted token signers configured"}))
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
		var (
			// a delegator that signs other users requests, has no fleet management feature access
			delegateSeedFile, delegateWTFile string

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
			delegateWTFile = filepath.Join(td, "delegate.jwt")
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
			if rand.Intn(10) <= 5 {
				cfg.TrustedTokenSigners = []ed25519.PublicKey{otherPubk, signerPubk}
			} else {
				cfg.TrustedTokenSigners = []ed25519.PublicKey{signerPubk, otherPubk}
			}

			// signs delegated requests
			delegatePubk, delegatePrik, err = iu.Ed25519KeyPairToFile(delegateSeedFile)
			Expect(err).ToNot(HaveOccurred())
			delegateToken, err := tokens.NewClientIDClaims("delegate", nil, "choria", nil, "", "Ginkgo Tests", time.Minute, &tokens.ClientPermissions{AuthenticationDelegator: true}, signerPubk)
			Expect(err).ToNot(HaveOccurred())
			Expect(delegateToken.Permissions.AuthenticationDelegator).To(BeTrue())
			delegateToken.PublicKey = hex.EncodeToString(delegatePubk)
			Expect(tokens.SaveAndSignTokenWithKeyFile(delegateToken, signerSeedFile, delegateWTFile, 0600)).To(Succeed())

			// a caller that needs delegation
			delegatedCallerPubk, _, err := iu.Ed25519KeyPairToFile(delegatedCallerSeedFile)
			Expect(err).ToNot(HaveOccurred())
			delegatedCallerToken, err := tokens.NewClientIDClaims("delegated_caller", nil, "choria", nil, "", "Ginkgo Tests", time.Minute, &tokens.ClientPermissions{SignedFleetManagement: true}, signerPubk)
			Expect(err).ToNot(HaveOccurred())
			Expect(delegatedCallerToken.Permissions.SignedFleetManagement).To(BeTrue())
			delegatedCallerToken.PublicKey = hex.EncodeToString(delegatedCallerPubk)
			Expect(tokens.SaveAndSignTokenWithKeyFile(delegatedCallerToken, signerSeedFile, delegatedCallerJWTFile, 0600)).To(Succeed())

			// caller that can sign itself
			callerPubk, callerPrik, err = iu.Ed25519KeyPairToFile(callerSeedFile)
			Expect(err).ToNot(HaveOccurred())
			callerToken, err := tokens.NewClientIDClaims("caller", nil, "choria", nil, "", "Ginkgo Tests", time.Minute, &tokens.ClientPermissions{FleetManagement: true}, signerPubk)
			Expect(err).ToNot(HaveOccurred())
			Expect(callerToken.Permissions.FleetManagement).To(BeTrue())
			callerToken.PublicKey = hex.EncodeToString(callerPubk)
			Expect(tokens.SaveAndSignTokenWithKeyFile(callerToken, signerSeedFile, callerJWTFile, 0600)).To(Succeed())

			// server token
			serverPubk, serverPrik, err = iu.Ed25519KeyPairToFile(serverSeedFile)
			Expect(err).ToNot(HaveOccurred())
			serverToken, err := tokens.NewServerClaims("example.net", []string{"choria"}, "choria", nil, nil, serverPubk, "ginkgo", time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(serverToken.Purpose).To(Equal(tokens.ServerPurpose))
			serverToken.PublicKey = hex.EncodeToString(serverPubk)
			Expect(tokens.SaveAndSignTokenWithKeyFile(serverToken, signerSeedFile, serverJWTFile, 0600)).To(Succeed())

			// a provisioner purpose token
			provToken, err := tokens.NewProvisioningClaims(true, true, "x", "x", "x", nil, "example.net", "", "", "ginkgo", time.Minute)
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
				Expect(logbuf).To(gbytes.Say("Could not verify signature by caller which requires authority delegation"))
			})

			It("Should deny client tokens that does not have fleet management access", func() {
				pub, err := os.ReadFile(delegateWTFile)
				Expect(err).ToNot(HaveOccurred())
				should, signer := prov.VerifySignatureBytes(nil, nil, pub)
				Expect(should).To(BeFalse())
				Expect(signer).To(Equal(""))
				Expect(logbuf).To(gbytes.Say("Could not verify signature by caller which does not have fleet management access"))
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
				should, signer := prov.VerifySignatureBytes(nil, nil, pub)
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
				Expect(logbuf).To(gbytes.Say("Token attempted to sign a request as delegator without required delegator permission"))
			})

			It("Should ensure the caller has fleet access", func() {
				delegate, err := os.ReadFile(delegateWTFile)
				Expect(err).ToNot(HaveOccurred())

				sig, err := iu.Ed25519Sign(delegatePrik, []byte("too many secrets"))
				Expect(err).ToNot(HaveOccurred())

				should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, delegate, delegate)
				Expect(should).To(BeFalse())
				Expect(signer).To(Equal(""))
				Expect(logbuf).To(gbytes.Say("Caller token can not be used without fleet management access"))
			})

			It("Should fail for server tokens", func() {
				delegate, err := os.ReadFile(delegateWTFile)
				Expect(err).ToNot(HaveOccurred())

				server, err := os.ReadFile(serverJWTFile)
				Expect(err).ToNot(HaveOccurred())

				sig, err := iu.Ed25519Sign(delegatePrik, []byte("too many secrets"))
				Expect(err).ToNot(HaveOccurred())

				should, signer := prov.VerifySignatureBytes([]byte("too many secrets"), sig, server, delegate)
				Expect(should).To(BeFalse())
				Expect(signer).To(Equal(""))
				Expect(logbuf).To(gbytes.Say("Could not load caller token using the same signer as the delegator: not a client token"))
			})

			It("Should support client tokens", func() {
				delegate, err := os.ReadFile(delegateWTFile)
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

			Expect(c.Certificates).To(HaveLen(0))
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
