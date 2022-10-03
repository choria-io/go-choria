// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/message"
	"github.com/choria-io/go-choria/protocol"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Protocol", func() {
	var j *[]byte
	var c *Framework
	var err error

	BeforeEach(func() {
		if j == nil {
			cfg := config.NewConfigForTests()
			cfg.DisableTLS = true
			c, err = NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			rm, err := message.NewMessage([]byte("ping"), "discovery", "mcollective", inter.RequestMessageType, nil, c)
			Expect(err).ToNot(HaveOccurred())

			req, err := c.NewRequestFromMessage(protocol.RequestV1, rm)
			Expect(err).ToNot(HaveOccurred())

			reply, err := message.NewMessage([]byte("pong"), "discovery", "mcollective", inter.ReplyMessageType, rm, c)
			Expect(err).ToNot(HaveOccurred())

			replyT, err := c.NewReplyTransportForMessage(reply, req)
			Expect(err).ToNot(HaveOccurred())

			js, err := replyT.JSON()
			Expect(err).ToNot(HaveOccurred())

			t := js
			j = &t
		}
	})
})
