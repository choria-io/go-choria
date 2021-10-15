// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package builddefaults

import (
	"testing"

	"github.com/choria-io/go-choria/build"
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
	)

	BeforeEach(func() {
		prov = &Resolver{}
	})

	Describe("Configure", func() {
		It("Should handle malformed jwt", func() {
			build.ProvisionJWTFile = "testdata/invalid.jwt"
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).To(MatchError("jwt parse error: token contains an invalid number of segments"))
		})

		It("Should detect missing auth token", func() {
			build.ProvisionJWTFile = "testdata/invalid_no_token.jwt"
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).To(MatchError("no auth token"))
		})

		It("Should detect missing url and srv domain", func() {
			build.ProvisionJWTFile = "testdata/invalid_no_url_or_srv.jwt"
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).To(MatchError("no srv domain or urls"))

		})

		It("Should detect both url and srv domain supplied", func() {
			build.ProvisionJWTFile = "testdata/invalid_url_and_srv.jwt"
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).To(MatchError("both srv domain and URLs supplied"))
		})

		It("Should set build properties for specific URL", func() {
			build.ProvisionJWTFile = "testdata/valid_url.jwt"
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).ToNot(HaveOccurred())
			Expect(build.ProvisionBrokerURLs).To(Equal("prov.example.net:4222"))
			Expect(build.ProvisionBrokerSRVDomain).To(Equal(""))
			Expect(build.ProvisionToken).To(Equal("secret"))
			Expect(build.ProvisionSecure).To(Equal("true"))
			Expect(build.ProvisionModeDefault).To(Equal("false"))
		})

		It("Should set build properties for specific SRV domain", func() {
			build.ProvisionJWTFile = "testdata/valid_srv.jwt"
			_, err := prov.setBuildBasedOnJWT()
			Expect(err).ToNot(HaveOccurred())
			Expect(build.ProvisionBrokerURLs).To(Equal(""))
			Expect(build.ProvisionBrokerSRVDomain).To(Equal("example.net"))
			Expect(build.ProvisionToken).To(Equal("secret"))
			Expect(build.ProvisionSecure).To(Equal("true"))
			Expect(build.ProvisionModeDefault).To(Equal("false"))
		})

		It("Should set provision mode default", func() {
			build.ProvisionJWTFile = "testdata/valid_prov_default.jwt"
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
