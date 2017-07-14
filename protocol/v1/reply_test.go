package v1

import (
	"time"

	"github.com/choria-io/go-choria/protocol"
	"github.com/tidwall/gjson"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reply", func() {
	It("should create the correct reply from a request", func() {
		request, _ := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		reply, _ := NewReply(request)

		reply.SetMessage("hello world")

		j, _ := reply.JSON()

		Expect(gjson.Get(j, "protocol").String()).To(Equal(protocol.ReplyV1))
		Expect(reply.Message()).To(Equal("hello world"))
		Expect(len(reply.RequestID())).To(Equal(32))
		Expect(reply.SenderID()).To(Equal("go.tests"))
		Expect(reply.Agent()).To(Equal("test"))
		Expect(reply.Time()).To(BeTemporally("~", time.Now(), time.Second))
	})
})
