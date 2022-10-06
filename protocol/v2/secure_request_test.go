// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/inter"
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
	var tech inter.SecurityTechnology
	var req protocol.Request
	var err error

	BeforeEach(func() {
		logrus.SetLevel(logrus.FatalLevel)
		mockctl = gomock.NewController(GinkgoT())
		security = imock.NewMockSecurityProvider(mockctl)

		security.EXPECT().BackingTechnology().DoAndReturn(func() inter.SecurityTechnology {
			return tech
		}).AnyTimes()

		req, err = NewRequest("ginkgo", "ginkgo.example.net", "up=ginkgo", 60, "1234", "choria")
		Expect(err).ToNot(HaveOccurred())
		req.SetMessage([]byte("hello"))

		tech = inter.SecurityTechnologyED25519JWT
		protocol.Secure = "true"
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("NewSecureRequest", func() {
		It("Should require the correct security technology", func() {
			tech = inter.SecurityTechnologyX509
			_, err := NewSecureRequest(nil, security)
			Expect(err).To(MatchError(ErrIncorrectProtocol))
		})

		It("Should support insecure operation", func() {
			protocol.Secure = "false"
			sreq, err := NewSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())
			Expect(sreq.(*SecureRequest).CallerJWT).To(Equal(""))
		})

		It("Should handle token lookup failures", func() {
			security.EXPECT().TokenBytes().Return(nil, fmt.Errorf("ginkgo"))

			sreq, err := NewSecureRequest(req, security)
			Expect(err).To(MatchError("ginkgo"))
			Expect(sreq).To(BeNil())
		})

		It("Should handle signing failures", func() {
			security.EXPECT().TokenBytes().Return([]byte("token"), nil)
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return(nil, fmt.Errorf("stub failure")).AnyTimes()

			sreq, err := NewSecureRequest(req, security)
			Expect(err).To(MatchError("stub failure"))
			Expect(sreq).To(BeNil())
		})

		It("Should produce a correct secure request", func() {
			security.EXPECT().TokenBytes().Return([]byte("token"), nil)
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()

			sreq, err := NewSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())

			r := sreq.(*SecureRequest)
			Expect(r.CallerJWT).To(Equal("token"))
			Expect(r.Signature).To(Equal([]byte("stub sig")))
			Expect(r.SignerJWT).To(Equal(""))
			Expect(r.MessageBody).To(ContainSubstring("io.choria.protocol.v2.request"))
		})
	})

	Describe("NewRemoteSignedSecureRequest", func() {
		It("Should require the correct security technology", func() {
			tech = inter.SecurityTechnologyX509
			_, err := NewRemoteSignedSecureRequest(nil, security)
			Expect(err).To(MatchError(ErrIncorrectProtocol))
		})

		Describe("Should support insecure or remote signing operation", func() {
			It("Should handle signing failures", func() {
				security.EXPECT().RemoteSignRequest(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("simulated failure"))
				_, err := NewRemoteSignedSecureRequest(req, security)
				Expect(err).To(MatchError("simulated failure"))
			})

			It("Should not call remote sign for the signing agent", func() {
				security.EXPECT().RemoteSignRequest(gomock.Any(), gomock.Any()).Times(0)

				// will call NewSecureRequest() and we have no expect on RemoteSignRequest()
				security.EXPECT().TokenBytes().Return([]byte("token"), nil)
				security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()

				req, err = NewRequest(protocol.RemoteSigningAgent, "ginkgo.example.net", "up=ginkgo", 60, "1234", "choria")
				Expect(err).ToNot(HaveOccurred())
				req.SetMessage([]byte("hello"))

				_, err := NewRemoteSignedSecureRequest(req, security)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("Should check the secure request is signed by a signer", func() {
			security.EXPECT().RemoteSignRequest(gomock.Any(), gomock.AssignableToTypeOf([]byte{})).DoAndReturn(func(_ context.Context, reqj []byte) ([]byte, error) {
				Expect(gjson.GetBytes(reqj, "agent").String()).To(Equal("ginkgo"))
				Expect(gjson.GetBytes(reqj, "protocol").String()).To(Equal(string(protocol.RequestV2)))

				security.EXPECT().TokenBytes().Return([]byte("token"), nil)
				security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()

				signed, err := NewSecureRequest(req, security)
				Expect(err).ToNot(HaveOccurred())

				signedj, err := signed.JSON()
				Expect(err).ToNot(HaveOccurred())

				return signedj, nil
			})

			sreq, err := NewRemoteSignedSecureRequest(req, security)
			Expect(err).To(MatchError("remote signer did not set a signer JWT"))
			Expect(sreq).To(BeNil())
		})

		It("Should produce a correct secure request", func() {
			security.EXPECT().RemoteSignRequest(gomock.Any(), gomock.AssignableToTypeOf([]byte{})).DoAndReturn(func(_ context.Context, reqj []byte) ([]byte, error) {
				Expect(gjson.GetBytes(reqj, "agent").String()).To(Equal("ginkgo"))
				Expect(gjson.GetBytes(reqj, "protocol").String()).To(Equal(string(protocol.RequestV2)))

				security.EXPECT().TokenBytes().Return([]byte("token"), nil)
				security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()

				signed, err := NewSecureRequest(req, security)
				Expect(err).ToNot(HaveOccurred())

				signed.(*SecureRequest).SignerJWT = "signer jwt"
				signedj, err := signed.JSON()
				Expect(err).ToNot(HaveOccurred())

				return signedj, nil
			})

			sreq, err := NewRemoteSignedSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())
			Expect(sreq.(*SecureRequest).SignerJWT).To(Equal("signer jwt"))
		})
	})

	Describe("NewSecureRequestFromTransport", func() {
		It("Should require the correct security technology", func() {
			tech = inter.SecurityTechnologyX509
			_, err := NewSecureRequestFromTransport(nil, security, false)
			Expect(err).To(MatchError(ErrIncorrectProtocol))
		})

		It("Should detect invalid payloads", func() {
			sr, err := NewSecureRequestFromTransport(&TransportMessage{Data: []byte("{}")}, security, false)
			Expect(err).To(MatchError(ErrInvalidJSON))
			Expect(sr).To(BeNil())
		})

		It("Should support skipping validation", func() {
			security.EXPECT().TokenBytes().Return([]byte("token"), nil)
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()
			security.EXPECT().VerifySignatureBytes(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, "").Times(0)

			tsreq, err := NewSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())
			t, err := NewTransportMessage("ginkgo.example.net")
			Expect(err).ToNot(HaveOccurred())
			Expect(t.SetRequestData(tsreq)).To(Succeed())

			sreq, err := NewSecureRequestFromTransport(t, security, true)
			Expect(err).ToNot(HaveOccurred())
			r := sreq.(*SecureRequest)
			Expect(r.CallerJWT).To(Equal("token"))
			Expect(r.Signature).To(Equal([]byte("stub sig")))
			Expect(r.SignerJWT).To(Equal(""))
			Expect(r.MessageBody).To(ContainSubstring("io.choria.protocol.v2.request"))
		})

		It("Should handle validation failures", func() {
			security.EXPECT().TokenBytes().Return([]byte("token"), nil)
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()
			security.EXPECT().VerifySignatureBytes(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, "").Times(1)

			tsreq, err := NewSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())
			t, err := NewTransportMessage("ginkgo.example.net")
			Expect(err).ToNot(HaveOccurred())
			Expect(t.SetRequestData(tsreq)).To(Succeed())

			sreq, err := NewSecureRequestFromTransport(t, security, false)
			Expect(err).To(MatchError("secure request messages created from Transport Message did not pass security validation"))
			Expect(sreq).To(BeNil())
		})

		It("Should validate and produce a correct secure request", func() {
			security.EXPECT().TokenBytes().Return([]byte("token"), nil)
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()
			security.EXPECT().VerifySignatureBytes(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, "signer.example.net").Times(1)
			security.EXPECT().ShouldAllowCaller(gomock.Any(), gomock.Any()).DoAndReturn(func(caller string, jwts ...[]byte) (bool, error) {
				Expect(caller).To(Equal("up=ginkgo"))
				return true, nil
			})

			tsreq, err := NewSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())
			t, err := NewTransportMessage("ginkgo.example.net")
			Expect(err).ToNot(HaveOccurred())
			Expect(t.SetRequestData(tsreq)).To(Succeed())

			sreq, err := NewSecureRequestFromTransport(t, security, false)
			Expect(err).ToNot(HaveOccurred())
			r := sreq.(*SecureRequest)
			Expect(r.CallerJWT).To(Equal("token"))
			Expect(r.Signature).To(Equal([]byte("stub sig")))
			Expect(r.SignerJWT).To(Equal(""))
			Expect(r.MessageBody).To(ContainSubstring("io.choria.protocol.v2.request"))
		})
	})

	Describe("SetMessage", func() {
		It("Should handle invalid requests", func() {
			sreq := &SecureRequest{security: security}
			err := sreq.SetMessage(&Request{})
			Expect(err).To(MatchError(ErrInvalidJSON))
			Expect(err.Error()).To(HavePrefix("could not JSON encode reply message"))
		})

		It("Should support insecure operation", func() {
			protocol.Secure = "false"
			sreq := &SecureRequest{security: security}
			err := sreq.SetMessage(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(sreq.Signature).To(Equal([]byte("insecure")))
		})

		It("Should sign the body and store it", func() {
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()
			sreq := &SecureRequest{security: security}
			err := sreq.SetMessage(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(sreq.Signature).To(Equal([]byte("stub sig")))
		})
	})

	Describe("Valid", func() {
		It("Should support insecure operation", func() {
			protocol.Secure = "false"
			sreq := &SecureRequest{}
			Expect(sreq.Valid()).To(BeTrue())
		})

		It("Should detect signature validation failures", func() {
			security.EXPECT().TokenBytes().Return([]byte("caller jwt"), nil).AnyTimes()
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()
			security.EXPECT().VerifySignatureBytes(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, "").Times(1)

			sreq, err := NewSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())
			Expect(sreq.Valid()).To(BeFalse())
		})

		It("Should handle disallowed callers", func() {
			security.EXPECT().TokenBytes().Return([]byte("caller jwt"), nil).AnyTimes()
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()
			security.EXPECT().VerifySignatureBytes(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, "ginkgo").Times(1)
			security.EXPECT().ShouldAllowCaller(gomock.Any(), gomock.Any()).Return(false, fmt.Errorf("simulated failure"))

			sreq, err := NewSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())
			Expect(sreq.Valid()).To(BeFalse())
		})

		It("Should do correct validations", func() {
			security.EXPECT().TokenBytes().Return([]byte("caller jwt"), nil).AnyTimes()
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()

			security.EXPECT().VerifySignatureBytes(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(body []byte, sig []byte, public ...[]byte) (bool, string) {
				Expect(body).To(ContainSubstring("io.choria.protocol.v2.request"))
				Expect(body).To(ContainSubstring(`"message":"aGVsbG8="`))
				Expect(sig).To(Equal([]byte("stub sig")))
				Expect(public).To(HaveLen(2))
				Expect(public[0]).To(Equal([]byte("caller jwt")))
				Expect(public[1]).To(Equal([]byte("signer jwt")))

				return true, "ginkgo"
			}).Times(1)

			security.EXPECT().RemoteSignRequest(gomock.Any(), gomock.AssignableToTypeOf([]byte{})).DoAndReturn(func(_ context.Context, reqj []byte) ([]byte, error) {
				signed, err := NewSecureRequest(req, security)
				Expect(err).ToNot(HaveOccurred())

				signed.(*SecureRequest).SignerJWT = "signer jwt"
				signedj, err := signed.JSON()
				Expect(err).ToNot(HaveOccurred())

				return signedj, nil
			}).AnyTimes()

			security.EXPECT().ShouldAllowCaller(gomock.Any(), gomock.Any()).DoAndReturn(func(caller string, public ...[]byte) (bool, error) {
				Expect(caller).To(Equal("up=ginkgo"))
				Expect(public).To(HaveLen(2))
				Expect(public[0]).To(Equal([]byte("caller jwt")))
				Expect(public[1]).To(Equal([]byte("signer jwt")))

				return false, nil
			}).Times(1)

			sreq, err := NewRemoteSignedSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())
			Expect(sreq.Valid()).To(BeTrue())
		})
	})

	Describe("IsValidJSON", func() {
		It("Should detect invalid JSON data", func() {
			sr := &SecureRequest{}
			err := sr.IsValidJSON([]byte("{}"))
			Expect(err).To(MatchError("supplied JSON document does not pass schema validation: missing properties: 'protocol', 'request', 'signature', 'caller'"))
		})

		It("Should accept valid JSON data", func() {
			security.EXPECT().TokenBytes().Return([]byte("token"), nil)
			security.EXPECT().SignBytes(gomock.AssignableToTypeOf([]byte{})).Return([]byte("stub sig"), nil).AnyTimes()

			sreq, err := NewSecureRequest(req, security)
			Expect(err).ToNot(HaveOccurred())

			j, err := sreq.JSON()
			Expect(err).ToNot(HaveOccurred())

			Expect(sreq.IsValidJSON(j)).ToNot(HaveOccurred())
		})
	})
})
