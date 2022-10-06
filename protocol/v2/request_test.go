// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"time"

	"github.com/choria-io/go-choria/protocol"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Request", func() {
	Describe("Filter", func() {
		It("Should report the correct filter", func() {
			r := Request{}
			f, filtered := r.Filter()
			Expect(filtered).To(BeFalse())
			Expect(f.Empty()).To(BeTrue())

			f = protocol.NewFilter()
			f.AddClassFilter("x")
			r.SetFilter(f)
			f, filtered = r.Filter()
			Expect(filtered).To(BeTrue())
			Expect(f.Empty()).To(BeFalse())
			Expect(f.ClassFilters()).To(Equal([]string{"x"}))
		})
	})

	Describe("SetMessage", func() {
		It("Should correctly set the message", func() {
			r := Request{}
			Expect(r.MessageBody).To(BeNil())
			r.SetMessage([]byte("hello world"))
			Expect(r.MessageBody).To(Equal([]byte("hello world")))
		})
	})

	Describe("NewRequestFromSecureRequest", func() {
		It("Should ensure the secure request is the compatible version", func() {
			r, err := NewRequestFromSecureRequest(&SecureRequest{Protocol: protocol.SecureRequestV1})
			Expect(err).To(MatchError("cannot create a version 2 SecureRequest from a choria:secure:request:1 SecureRequest"))
			Expect(r).To(BeNil())
		})

		It("Should validate the secure request and fail for invalid ones", func() {
			r, err := NewRequestFromSecureRequest(&SecureRequest{Protocol: protocol.SecureRequestV2, MessageBody: []byte("{}")})
			Expect(err).To(MatchError("the JSON body from the SecureRequest is not a valid Request message: supplied JSON document does not pass schema validation: missing properties: 'protocol', 'message', 'id', 'sender', 'caller', 'collective', 'agent', 'ttl', 'time'"))
			Expect(r).To(BeNil())
		})

		It("Should create a valid request", func() {
			req, err := NewRequest("ginkgo", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "choria")
			req.SetMessage([]byte("hello world"))
			Expect(err).ToNot(HaveOccurred())
			j, err := req.JSON()
			Expect(err).ToNot(HaveOccurred())

			r, err := NewRequestFromSecureRequest(&SecureRequest{Protocol: protocol.SecureRequestV2, MessageBody: j})
			Expect(err).ToNot(HaveOccurred())
			Expect(r.RequestID()).To(Equal("a2f0ca717c694f2086cfa81b6c494648"))
		})
	})

	Describe("RecordNetworkHop", func() {
		It("Should record the hop correctly", func() {
			r := &Request{}
			Expect(r.ReqEnvelope.seenBy).To(HaveLen(0))
			r.RecordNetworkHop("ginkgo.in", "server1", "ginkgo.out")
			Expect(r.ReqEnvelope.seenBy).To(HaveLen(1))
			r.RecordNetworkHop("ginkgo.in", "server2", "ginkgo.out")
			Expect(r.ReqEnvelope.seenBy).To(HaveLen(2))
			Expect(r.ReqEnvelope.seenBy[0]).To(Equal([3]string{"ginkgo.in", "server1", "ginkgo.out"}))
		})
	})

	Describe("NetworkHops", func() {
		It("Should report the correct hops", func() {
			r := &Request{}
			r.RecordNetworkHop("ginkgo.in", "server1", "ginkgo.out")
			r.RecordNetworkHop("ginkgo.in", "server2", "ginkgo.out")
			Expect(r.ReqEnvelope.seenBy).To(HaveLen(2))
			hops := r.NetworkHops()
			Expect(hops).To(HaveLen(2))
			Expect(hops[0]).To(Equal([3]string{"ginkgo.in", "server1", "ginkgo.out"}))
		})
	})

	Describe("Federation", func() {
		Describe("SetFederationRequestID", func() {
			It("Should set the id correctly", func() {
				r := Request{}
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
				r := Request{}
				r.SetFederationRequestID("123")
				Expect(r.IsFederated()).To(BeTrue())
				r.SetUnfederated()
				Expect(r.IsFederated()).To(BeFalse())
			})
		})

		Describe("SetFederationTargets", func() {
			It("Should set the federation targets correctly", func() {
				r := &Request{}
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
				r := Request{}
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
				r := Request{}
				Expect(r.IsFederated()).To(BeFalse())
				r.SetFederationReplyTo("reply.to")
				Expect(r.IsFederated()).To(BeTrue())
			})
		})
	})

	Describe("NewRequest", func() {
		It("Should validate requests against the schema", func() {
			req, err := NewRequest("", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
			Expect(err).ToNot(HaveOccurred())
			_, err = req.JSON()
			Expect(err).To(MatchError(ErrInvalidJSON))
		})

		It("Should construct the correct request", func() {
			request, err := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
			Expect(err).ToNot(HaveOccurred())
			filter, filtered := request.Filter()

			request.SetMessage([]byte("hello world"))

			j, err := request.JSON()
			Expect(err).ToNot(HaveOccurred())

			Expect(protocol.VersionFromJSON(j)).To(Equal(protocol.RequestV2))
			Expect(request.Version()).To(Equal(protocol.RequestV2))
			Expect(request.Message()).To(Equal([]byte("hello world")))
			Expect(len(request.RequestID())).To(Equal(32))
			Expect(request.SenderID()).To(Equal("go.tests"))
			Expect(request.CallerID()).To(Equal("choria=test"))
			Expect(request.Collective()).To(Equal("mcollective"))
			Expect(request.Agent()).To(Equal("test"))
			Expect(request.TTL()).To(Equal(120))
			Expect(request.Time()).To(BeTemporally("~", time.Now(), time.Second))
			Expect(filtered).To(BeFalse())
			Expect(filter.Empty()).To(BeTrue())

			filter.AddAgentFilter("rpcutil")
			filter, filtered = request.Filter()
			Expect(filtered).To(BeTrue())
			Expect(filter).ToNot(BeNil())

			filter.AddAgentFilter("other")
			filter, filtered = request.Filter()
			Expect(filtered).To(BeTrue())
			Expect(filter).ToNot(BeNil())
		})
	})
})
