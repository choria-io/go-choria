package choria

import (
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-protocol/protocol"
	. "github.com/onsi/ginkgo"
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

			rm, err := NewMessage("ping", "discovery", "mcollective", "request", nil, c)
			Expect(err).ToNot(HaveOccurred())

			req, err := c.NewRequestFromMessage(protocol.RequestV1, rm)
			Expect(err).ToNot(HaveOccurred())

			reply, err := NewMessage("pong", "discovery", "mcollective", "reply", rm, c)
			Expect(err).ToNot(HaveOccurred())

			replyT, err := c.NewReplyTransportForMessage(reply, req)
			Expect(err).ToNot(HaveOccurred())

			js, err := replyT.JSON()
			Expect(err).ToNot(HaveOccurred())

			t := []byte(js)
			j = &t
		}
	})
})
