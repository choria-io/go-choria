// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package submission

import (
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"strconv"
)

var _ = Describe("Message", func() {
	Describe("NatsMessage", func() {
		It("Check max tried", func() {
			msg := newMessage("ginkgo")
			msg.MaxTries = 10
			msg.Tries = 10
			msg.Subject = "x"
			msg.Priority = 2
			msg.Payload = []byte("hello world")
			msg.Headers = map[string]string{"Hello": "World"}

			_, err := msg.NatsMessage("x", "", "")
			Expect(err).To(MatchError("message reached maximum tries"))

			msg.Tries = 1

			nm, err := msg.NatsMessage("x", "", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(nm.Header).To(Equal(nats.Header{
				"Nats-Msg-Id":     []string{msg.ID},
				"Choria-Priority": []string{"2"},
				"Choria-Created":  []string{strconv.Itoa(int(msg.Created.UnixNano()))},
				"Choria-Sender":   []string{"ginkgo"},
				"Choria-Tries":    []string{"1"},
				"Hello":           []string{"World"}}))
		})
	})
})
