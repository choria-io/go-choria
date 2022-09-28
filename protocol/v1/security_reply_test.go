// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"crypto/sha256"
	"encoding/base64"

	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/protocol"
	"github.com/golang/mock/gomock"
	"github.com/tidwall/gjson"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SecureReply", func() {
	var mockctl *gomock.Controller
	var security *imock.MockSecurityProvider

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		security = imock.NewMockSecurityProvider(mockctl)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	It("Should create valid replies", func() {
		request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		request.SetMessage(`{"test":1}`)

		reply, err := NewReply(request, "testing")
		Expect(err).ToNot(HaveOccurred())

		rj, err := reply.JSON()
		Expect(err).ToNot(HaveOccurred())

		sha := sha256.Sum256([]byte(rj))

		security.EXPECT().ChecksumBytes([]byte(rj)).Return(sha[:]).AnyTimes()

		sreply, _ := NewSecureReply(reply, security)
		sj, err := sreply.JSON()
		Expect(err).ToNot(HaveOccurred())

		Expect(gjson.Get(sj, "protocol").String()).To(Equal(protocol.SecureReplyV1))
		Expect(gjson.Get(sj, "message").String()).To(Equal(rj))
		Expect(gjson.Get(sj, "hash").String()).To(Equal(base64.StdEncoding.EncodeToString(sha[:])))
		Expect(sreply.Valid()).To(BeTrue())
	})
})
