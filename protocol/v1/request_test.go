package v1

import (
	"time"

	"github.com/choria-io/go-protocol/protocol"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

var _ = Describe("Request", func() {
	It("Should construct the correct request", func() {
		request, _ := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		filter, filtered := request.Filter()

		request.SetMessage("hello world")

		j, _ := request.JSON()

		Expect(gjson.Get(j, "protocol").String()).To(Equal(protocol.RequestV1))
		Expect(request.Message()).To(Equal("hello world"))
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
	})

	Measure("Request creation time", func(b Benchmarker) {
		runtime := b.Time("runtime", func() {
			request, _ := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
			request.SetMessage(`{"hello":"world"}`)
		})

		Expect(runtime.Nanoseconds()).Should(BeNumerically("<", 100000))
	}, 1000)
})
