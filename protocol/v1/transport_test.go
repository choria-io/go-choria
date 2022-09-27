// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
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
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	It("Should support reply data", func() {
		security.EXPECT().ChecksumBytes(gomock.Any()).Return([]byte("stub checksum")).AnyTimes()

		request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		request.SetMessage([]byte(`{"message":1}`))
		reply, _ := NewReply(request, "testing")
		sreply, _ := NewSecureReply(reply, security)
		treply, _ := NewTransportMessage("rip.mcollective")
		err := treply.SetReplyData(sreply)
		Expect(err).ToNot(HaveOccurred())

		sj, err := sreply.JSON()
		Expect(err).ToNot(HaveOccurred())

		j, err := treply.JSON()
		Expect(err).ToNot(HaveOccurred())

		Expect(gjson.GetBytes(j, "protocol").String()).To(Equal(protocol.TransportV1))
		Expect(gjson.GetBytes(j, "headers.mc_sender").String()).To(Equal("rip.mcollective"))

		d, err := treply.Message()
		Expect(err).ToNot(HaveOccurred())

		Expect(d).To(Equal(sj))
	})

	It("Should support request data", func() {
		security.EXPECT().PublicCertBytes().Return([]byte("stub cert"), nil).AnyTimes()
		security.EXPECT().SignBytes(gomock.Any()).Return([]byte("stub sig"), nil).AnyTimes()

		request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		request.SetMessage([]byte(`{"message":1}`))
		srequest, _ := NewSecureRequest(request, security)
		trequest, _ := NewTransportMessage("rip.mcollective")
		trequest.SetRequestData(srequest)

		sj, _ := srequest.JSON()
		j, _ := trequest.JSON()

		Expect(gjson.GetBytes(j, "protocol").String()).To(Equal(protocol.TransportV1))
		Expect(gjson.GetBytes(j, "headers.mc_sender").String()).To(Equal("rip.mcollective"))

		d, err := trequest.Message()
		Expect(err).ToNot(HaveOccurred())

		Expect(d).To(Equal(sj))
	})

	It("Should support creation from JSON data", func() {
		protocol.ClientStrictValidation = true
		defer func() { protocol.ClientStrictValidation = false }()

		security.EXPECT().PublicCertBytes().Return([]byte("stub cert"), nil).AnyTimes()
		security.EXPECT().SignBytes(gomock.Any()).Return([]byte("stub sig"), nil).AnyTimes()

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
		Expect(err).To(MatchError("supplied JSON document is not a valid Transport message: (root): data is required, (root): headers is required, protocol: Invalid type. Expected: string, given: integer"))
	})
})
