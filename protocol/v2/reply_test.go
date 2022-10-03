// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"time"

	"github.com/choria-io/go-choria/protocol"
	"github.com/tidwall/gjson"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reply", func() {
	It("should create the correct reply from a request", func() {
		request, err := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		Expect(err).ToNot(HaveOccurred())
		request.SetMessage([]byte("hello"))
		reply, err := NewReply(request, "testing")
		Expect(err).ToNot(HaveOccurred())

		reply.SetMessage([]byte("hello world"))

		j, _ := reply.JSON()

		Expect(gjson.GetBytes(j, "protocol").String()).To(Equal(protocol.ReplyV2))
		Expect(reply.Message()).To(Equal([]byte("hello world")))
		Expect(len(reply.RequestID())).To(Equal(32))
		Expect(reply.SenderID()).To(Equal("testing"))
		Expect(reply.Agent()).To(Equal("test"))
		Expect(reply.Time()).To(BeTemporally("~", time.Now(), time.Second))
	})
})
