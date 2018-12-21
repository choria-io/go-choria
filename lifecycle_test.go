package lifecycle

import (
	"os"
	"testing"

	gomock "github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestChoria(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Events")
}

var _ = Describe("Events", func() {
	var (
		mockctl *gomock.Controller
		conn    *MockPublishConnector
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		conn = NewMockPublishConnector(mockctl)
		mockTime = 1535106973
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("New", func() {
		It("Should create a valid event", func() {
			event, err := New(Startup, Component("ginkgo"))
			Expect(err).ToNot(HaveOccurred())
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.Type()).To(Equal(Startup))
		})
	})

	Describe("EventTypeNames", func() {
		It("Should list all known types", func() {
			Expect(EventTypeNames()).To(Equal([]string{"alive", "provisioned", "shutdown", "startup"}))
		})
	})

	Describe("NewFromJSON", func() {
		It("Should handle messages without protocols", func() {
			_, err := NewFromJSON([]byte("{}"))
			Expect(err).To(MatchError("no protocol field present"))
		})

		It("Should handle invalid protocols", func() {
			_, err := NewFromJSON([]byte(`{"protocol":"fail"}`))
			Expect(err).To(MatchError("invalid protocol 'fail' received"))
		})

		It("Should handle unknown event types", func() {
			_, err := NewFromJSON([]byte(`{"protocol":"choria:lifecycle:unknown:1"}`))
			Expect(err).To(MatchError("unknown protocol 'choria:lifecycle:unknown:1' received"))
		})
	})

	Describe("PublishEvent", func() {
		It("Should publish the event to the right destination", func() {
			event, err := New(Startup, Component("ginkgo"), Version("1.2.3"), Identity("ginkgo.example.net"))
			Expect(err).ToNot(HaveOccurred())
			mockTime = 1535106973
			Expect(err).ToNot(HaveOccurred())
			conn.EXPECT().PublishRaw("choria.lifecycle.event.startup.ginkgo", []byte(`{"protocol":"choria:lifecycle:startup:1","identity":"ginkgo.example.net","component":"ginkgo","timestamp":1535106973,"version":"1.2.3"}`))
			PublishEvent(event, conn)
		})
	})
})
