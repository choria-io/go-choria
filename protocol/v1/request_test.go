// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"time"

	"github.com/choria-io/go-choria/protocol"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Request", func() {
	It("Should construct the correct request", func() {
		request, _ := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		filter, filtered := request.Filter()

		request.SetMessage([]byte("hello world"))

		j, _ := request.JSON()

		Expect(protocol.VersionFromJSON(j)).To(Equal(protocol.RequestV1))
		Expect(request.Message()).To(Equal([]byte("hello world")))
		Expect(request.RequestID()).To(HaveLen(32))
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
