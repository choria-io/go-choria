// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/protocol"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

var _ = Describe("TransportMessage", func() {
	var mockctl *gomock.Controller
	var security *imock.MockSecurityProvider

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		security = imock.NewMockSecurityProvider(mockctl)
		security.EXPECT().BackingTechnology().Return(inter.SecurityTechnologyED25519JWT)
		security.EXPECT().ShouldSignReplies().Return(true).AnyTimes()
		security.EXPECT().ChecksumBytes(gomock.Any()).Return([]byte("stub checksum")).AnyTimes()
		security.EXPECT().TokenBytes().Return([]byte("stub token bytes"), nil).AnyTimes()
		security.EXPECT().SignBytes(gomock.Any()).Return([]byte("stub signature"), nil).AnyTimes()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	It("Should support reply data", func() {
		request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		request.SetMessage([]byte(`{"message":1}`))
		reply, err := NewReply(request, "testing")
		Expect(err).ToNot(HaveOccurred())
		sreply, err := NewSecureReply(reply, security)
		Expect(err).ToNot(HaveOccurred())
		treply, err := NewTransportMessage("rip.mcollective")
		Expect(err).ToNot(HaveOccurred())
		err = treply.SetReplyData(sreply)
		Expect(err).ToNot(HaveOccurred())

		sj, err := sreply.JSON()
		Expect(err).ToNot(HaveOccurred())

		j, err := treply.JSON()
		Expect(err).ToNot(HaveOccurred())

		Expect(protocol.VersionFromJSON(j)).To(Equal(protocol.TransportV2))
		Expect(gjson.GetBytes(j, "headers.sender").String()).To(Equal("rip.mcollective"))

		d, err := treply.Message()
		Expect(err).ToNot(HaveOccurred())

		Expect(d).To(Equal(sj))
	})

	It("Should support request data", func() {
		request, err := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		Expect(err).ToNot(HaveOccurred())
		request.SetMessage([]byte(`{"message":1}`))
		srequest, err := NewSecureRequest(request, security)
		Expect(err).ToNot(HaveOccurred())
		trequest, err := NewTransportMessage("rip.mcollective")
		Expect(err).ToNot(HaveOccurred())
		trequest.SetRequestData(srequest)

		sj, err := srequest.JSON()
		Expect(err).ToNot(HaveOccurred())
		j, err := trequest.JSON()
		Expect(err).ToNot(HaveOccurred())

		Expect(protocol.VersionFromJSON(j)).To(Equal(protocol.TransportV2))
		Expect(gjson.GetBytes(j, "headers.sender").String()).To(Equal("rip.mcollective"))

		d, err := trequest.Message()
		Expect(err).ToNot(HaveOccurred())

		Expect(d).To(Equal(sj))
	})

	It("Should support creation from JSON data", func() {
		protocol.ClientStrictValidation = true
		defer func() { protocol.ClientStrictValidation = false }()

		request, err := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		Expect(err).ToNot(HaveOccurred())
		request.SetMessage([]byte("hello world"))
		srequest, err := NewSecureRequest(request, security)
		Expect(err).ToNot(HaveOccurred())
		trequest, err := NewTransportMessage("rip.mcollective")
		Expect(err).ToNot(HaveOccurred())

		Expect(trequest.SetRequestData(srequest)).To(Succeed())

		j, err := trequest.JSON()
		Expect(err).ToNot(HaveOccurred())

		_, err = NewTransportFromJSON(j)
		Expect(err).ToNot(HaveOccurred())

		_, err = NewTransportFromJSON([]byte(`{"protocol": 1}`))
		Expect(err).To(MatchError("supplied JSON document does not pass schema validation: missing properties: 'data', 'headers', /protocol: expected string, but got number"))
	})
})
