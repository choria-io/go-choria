// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/base64"
	"errors"
	"os"

	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/protocol"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

var _ = Describe("SecureRequest", func() {
	var mockctl *gomock.Controller
	var security *imock.MockSecurityProvider
	var pub []byte

	BeforeEach(func() {
		logrus.SetLevel(logrus.FatalLevel)
		mockctl = gomock.NewController(GinkgoT())
		security = imock.NewMockSecurityProvider(mockctl)

		protocol.Secure = "true"

		pub, _ = os.ReadFile("testdata/ssl/certs/rip.mcollective.pem")
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	It("Should support insecure mode", func() {
		security.EXPECT().PublicCertBytes().Return([]byte{}, errors.New("simulated")).AnyTimes()

		protocol.Secure = "false"

		r, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		r.SetMessage(`{"test":1}`)
		rj, err := r.JSON()
		Expect(err).ToNot(HaveOccurred())

		security.EXPECT().SignBytes(gomock.Any()).Times(0)

		sr, err := NewSecureRequest(r, security)
		Expect(err).ToNot(HaveOccurred())

		sj, err := sr.JSON()
		Expect(err).ToNot(HaveOccurred())

		Expect(gjson.Get(sj, "protocol").String()).To(Equal(protocol.SecureRequestV1))
		Expect(gjson.Get(sj, "message").String()).To(Equal(rj))
		Expect(gjson.Get(sj, "pubcert").String()).To(Equal("insecure"))
		Expect(gjson.Get(sj, "signature").String()).To(Equal("insecure"))
	})

	It("Should create a valid SecureRequest", func() {
		security.EXPECT().PublicCertBytes().Return(pub, nil).AnyTimes()

		r, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		r.SetMessage(`{"test":1}`)
		rj, err := r.JSON()
		Expect(err).ToNot(HaveOccurred())

		security.EXPECT().SignBytes([]byte(rj)).Return([]byte("stub.sig"), nil)

		sr, err := NewSecureRequest(r, security)
		Expect(err).ToNot(HaveOccurred())

		sj, err := sr.JSON()
		Expect(err).ToNot(HaveOccurred())

		Expect(gjson.Get(sj, "protocol").String()).To(Equal(protocol.SecureRequestV1))
		Expect(gjson.Get(sj, "message").String()).To(Equal(rj))
		Expect(gjson.Get(sj, "pubcert").String()).To(Equal(string(pub)))
		Expect(gjson.Get(sj, "signature").String()).To(Equal(base64.StdEncoding.EncodeToString([]byte("stub.sig"))))
	})
})
