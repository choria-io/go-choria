// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package builddefaults

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/tokens"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDefault(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provtarget/Default")
}

var _ = Describe("Default", func() {
	var (
		prov *Resolver
		td   string
		err  error
	)

	createToken := func(claims *tokens.ProvisioningClaims, td string) string {
		t, err := tokens.SignTokenWithKeyFile(claims, "testdata/signer-key.pem")
		Expect(err).ToNot(HaveOccurred())

		out := filepath.Join(td, "token.jwt")
		err = os.WriteFile(out, []byte(t), 0600)
		Expect(err).ToNot(HaveOccurred())

		return out
	}

	BeforeEach(func() {
		td, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		prov = &Resolver{}
	})

	AfterEach(func() {
		os.RemoveAll(td)
	})

	Describe("Configure", func() {
		It("Should handle malformed jwt", func() {
			build.ProvisionJWTFile = "testdata/invalid.jwt"
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).To(MatchError("token contains an invalid number of segments"))
		})

		It("Should detect missing auth token", func() {
			build.ProvisionJWTFile = createToken(&tokens.ProvisioningClaims{
				StandardClaims: tokens.StandardClaims{
					Purpose: tokens.ProvisioningPurpose,
				},
			}, td)
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).To(MatchError("no auth token"))
		})

		It("Should detect missing url and srv domain", func() {
			build.ProvisionJWTFile = createToken(&tokens.ProvisioningClaims{
				Token: "x",
				StandardClaims: tokens.StandardClaims{
					Purpose: tokens.ProvisioningPurpose,
				},
			}, td)

			_, err := prov.setBuildBasedOnJWT()
			Expect(err).To(MatchError("no srv domain or urls"))

		})

		It("Should detect both url and srv domain supplied", func() {
			build.ProvisionJWTFile = createToken(&tokens.ProvisioningClaims{
				Token:     "x",
				URLs:      "nats://example.net:4222",
				SRVDomain: "example.net",
				StandardClaims: tokens.StandardClaims{
					Purpose: tokens.ProvisioningPurpose,
				},
			}, td)
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).To(MatchError("both srv domain and URLs supplied"))
		})

		It("Should set build properties for specific URL", func() {
			build.ProvisionJWTFile = createToken(&tokens.ProvisioningClaims{
				Token:  "secret",
				URLs:   "prov.example.net:4222",
				Secure: true,
				StandardClaims: tokens.StandardClaims{
					Purpose: tokens.ProvisioningPurpose,
				},
			}, td)
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).ToNot(HaveOccurred())
			Expect(build.ProvisionBrokerURLs).To(Equal("prov.example.net:4222"))
			Expect(build.ProvisionBrokerSRVDomain).To(Equal(""))
			Expect(build.ProvisionToken).To(Equal("secret"))
			Expect(build.ProvisionSecure).To(Equal("true"))
			Expect(build.ProvisionModeDefault).To(Equal("false"))
		})

		It("Should set build properties for specific SRV domain", func() {
			build.ProvisionJWTFile = createToken(&tokens.ProvisioningClaims{
				Token:     "secret",
				SRVDomain: "example.net",
				Secure:    true,
				StandardClaims: tokens.StandardClaims{
					Purpose: tokens.ProvisioningPurpose,
				},
			}, td)
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).ToNot(HaveOccurred())
			Expect(build.ProvisionBrokerURLs).To(Equal(""))
			Expect(build.ProvisionBrokerSRVDomain).To(Equal("example.net"))
			Expect(build.ProvisionToken).To(Equal("secret"))
			Expect(build.ProvisionSecure).To(Equal("true"))
			Expect(build.ProvisionModeDefault).To(Equal("false"))
		})

		It("Should set provision mode default", func() {
			build.ProvisionJWTFile = createToken(&tokens.ProvisioningClaims{
				Token:       "secret",
				URLs:        "prov.example.net:4222",
				Secure:      true,
				ProvDefault: true,
				StandardClaims: tokens.StandardClaims{
					Purpose: tokens.ProvisioningPurpose,
				},
			}, td)
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).ToNot(HaveOccurred())
			Expect(build.ProvisionBrokerURLs).To(Equal("prov.example.net:4222"))
			Expect(build.ProvisionBrokerSRVDomain).To(Equal(""))
			Expect(build.ProvisionToken).To(Equal("secret"))
			Expect(build.ProvisionSecure).To(Equal("true"))
			Expect(build.ProvisionModeDefault).To(Equal("true"))
		})
	})
})
