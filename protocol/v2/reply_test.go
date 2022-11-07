// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"time"

	"github.com/choria-io/go-choria/protocol"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reply", func() {
	BeforeEach(func() {
		protocol.ClientStrictValidation = false
	})

	Describe("NewReplyFromSecureReply", func() {
		It("Should check the version secure reply", func() {
			r, err := NewReplyFromSecureReply(&SecureReply{Protocol: protocol.SecureReplyV1})
			Expect(err).To(MatchError("cannot create a version 2 Reply from a choria:secure:reply:1 SecureReply"))
			Expect(r).To(BeNil())
		})

		It("Should validate the message", func() {
			r, err := NewReplyFromSecureReply(&SecureReply{Protocol: protocol.SecureReplyV2, MessageBody: []byte(`{"x":"y"}`)})
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())

			protocol.ClientStrictValidation = true

			r, err = NewReplyFromSecureReply(&SecureReply{Protocol: protocol.SecureReplyV2, MessageBody: []byte(`{"x":"y"}`)})
			Expect(err).To(MatchError(ErrInvalidJSON))
			Expect(r).To(BeNil())
		})

		It("Should correctly create a reply", func() {
			request, err := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
			Expect(err).ToNot(HaveOccurred())
			request.SetMessage([]byte("hello world"))
			reply, err := NewReply(request, request.SenderID())
			Expect(err).ToNot(HaveOccurred())

			j, err := reply.JSON()
			Expect(err).ToNot(HaveOccurred())

			reply, err = NewReplyFromSecureReply(&SecureReply{Protocol: protocol.SecureReplyV2, MessageBody: j})
			Expect(err).ToNot(HaveOccurred())
			Expect(reply.SenderID()).To(Equal("go.tests"))
		})
	})

	Describe("RecordNetworkHop", func() {
		It("Should record the hop correctly", func() {
			r := &Reply{}
			Expect(r.seenBy).To(HaveLen(0))
			r.RecordNetworkHop("ginkgo.in", "server1", "ginkgo.out")
			Expect(r.seenBy).To(HaveLen(1))
			r.RecordNetworkHop("ginkgo.in", "server2", "ginkgo.out")
			Expect(r.seenBy).To(HaveLen(2))
			Expect(r.seenBy[0]).To(Equal([3]string{"ginkgo.in", "server1", "ginkgo.out"}))
		})
	})

	Describe("NetworkHops", func() {
		It("Should report the correct hops", func() {
			r := &Reply{}
			r.RecordNetworkHop("ginkgo.in", "server1", "ginkgo.out")
			r.RecordNetworkHop("ginkgo.in", "server2", "ginkgo.out")
			Expect(r.seenBy).To(HaveLen(2))
			hops := r.NetworkHops()
			Expect(hops).To(HaveLen(2))
			Expect(hops[0]).To(Equal([3]string{"ginkgo.in", "server1", "ginkgo.out"}))
		})
	})

	Describe("Federation", func() {
		Describe("SetFederationRequestID", func() {
			It("Should set the id correctly", func() {
				r := Reply{}
				id, federated := r.FederationRequestID()
				Expect(federated).To(BeFalse())
				Expect(id).To(Equal(""))

				r.SetFederationRequestID("123")
				id, federated = r.FederationRequestID()
				Expect(federated).To(BeTrue())
				Expect(id).To(Equal("123"))
			})
		})

		Describe("SetUnfederated", func() {
			It("Should correctly unfederate the message", func() {
				r := Reply{}
				r.SetFederationRequestID("123")
				Expect(r.IsFederated()).To(BeTrue())
				r.SetUnfederated()
				Expect(r.IsFederated()).To(BeFalse())
			})
		})

		Describe("SetFederationTargets", func() {
			It("Should set the federation targets correctly", func() {
				r := &Reply{}
				t, federated := r.FederationTargets()
				Expect(federated).To(BeFalse())
				Expect(t).To(HaveLen(0))
				r.SetFederationTargets([]string{"1", "2"})
				t, federated = r.FederationTargets()
				Expect(federated).To(BeTrue())
				Expect(t).To(Equal([]string{"1", "2"}))
			})
		})

		Describe("SetFederationReplyTo", func() {
			It("Should set the federation reply correctly", func() {
				r := Reply{}
				rt, federated := r.FederationReplyTo()
				Expect(federated).To(BeFalse())
				Expect(rt).To(Equal(""))

				r.SetFederationReplyTo("reply.to")
				rt, federated = r.FederationReplyTo()
				Expect(federated).To(BeTrue())
				Expect(rt).To(Equal("reply.to"))
			})
		})

		Describe("IsFederated", func() {
			It("Should report correctly", func() {
				r := Reply{}
				Expect(r.IsFederated()).To(BeFalse())
				r.SetFederationReplyTo("reply.to")
				Expect(r.IsFederated()).To(BeTrue())
			})
		})
	})

	Describe("NewReply", func() {
		It("should create the correct reply from a request", func() {
			request, err := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
			Expect(err).ToNot(HaveOccurred())
			request.SetMessage([]byte("hello world"))
			reply, err := NewReply(request, "testing")
			Expect(err).ToNot(HaveOccurred())

			reply.SetMessage([]byte("hello world"))
			j, _ := reply.JSON()

			Expect(protocol.VersionFromJSON(j)).To(Equal(protocol.ReplyV2))
			Expect(reply.Version()).To(Equal(protocol.ReplyV2))
			Expect(reply.Message()).To(Equal([]byte("hello world")))
			Expect(len(reply.RequestID())).To(Equal(32))
			Expect(reply.SenderID()).To(Equal("testing"))
			Expect(reply.Agent()).To(Equal("test"))
			Expect(reply.Time()).To(BeTemporally("~", time.Now(), time.Second))
		})
	})
})
