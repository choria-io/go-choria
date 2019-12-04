package lifecycle

import (
	gomock "github.com/golang/mock/gomock"
	"io/ioutil"
	"os"
	"testing"

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

	Describe("EventFormatFromJSON", func() {
		It("Should detect choria format", func() {
			Expect(EventFormatFromJSON([]byte("{}"))).To(Equal(UnknownFormat))
			Expect(EventFormatFromJSON([]byte(`{"protocol":"io.choria.lifecycle.v1.unknown"}`))).To(Equal(ChoriaFormat))
			Expect(EventFormatFromJSON([]byte(`{"protocol":"other"}`))).To(Equal(UnknownFormat))

		})

		It("Should detect cloudevent format", func() {
			Expect(EventFormatFromJSON([]byte(`{"specversion":"1.0", "source":"io.choria.lifecycle"}`))).To(Equal(CloudEventV1Format))
			Expect(EventFormatFromJSON([]byte(`{"specversion":"1.0", "source":"message/other"}`))).To(Equal(UnknownFormat))
			Expect(EventFormatFromJSON([]byte(`{"specversion":"0.1", "source":"message/io.choria.lifecycle"}`))).To(Equal(UnknownFormat))

		})
	})

	Describe("NewFromJSON", func() {
		Context("Choria Format", func() {
			It("Should handle invalid protocols", func() {
				_, err := NewFromJSON([]byte(`{"protocol":"fail"}`))
				Expect(err).To(MatchError("unsupported event format"))
			})

			It("Should handle unknown event types", func() {
				_, err := NewFromJSON([]byte(`{"protocol":"io.choria.lifecycle.v1.unknown"}`))
				Expect(err).To(MatchError("unknown protocol 'io.choria.lifecycle.v1.unknown' received"))
			})

			It("Should handle correctly formatted events", func() {
				j, err := ioutil.ReadFile("testdata/choriaFormatShutdown.json")
				Expect(err).ToNot(HaveOccurred())
				event, err := NewFromJSON(j)
				Expect(err).ToNot(HaveOccurred())
				Expect(event.Type()).To(Equal(Shutdown))
				Expect(event.Format()).To(Equal(ChoriaFormat))
			})
		})

		Context("CloudEvents Format", func() {
			It("Should handle correctly formatted events", func() {
				j, err := ioutil.ReadFile("testdata/cloudEventFormatShutdown.json")
				Expect(err).ToNot(HaveOccurred())
				event, err := NewFromJSON(j)
				Expect(err).ToNot(HaveOccurred())
				Expect(event.Type()).To(Equal(Shutdown))
				Expect(event.Component()).To(Equal("server"))
				Expect(event.Format()).To(Equal(CloudEventV1Format))
			})
		})
	})

	Describe("PublishEvent", func() {
		It("Should publish the event to the right destination", func() {
			mockTime = 1535106973
			mockID = "01e72410-d734-4611-9485-8c6a2dd2579b"

			event, err := New(Startup, Component("ginkgo"), Version("1.2.3"), Identity("ginkgo.example.net"))
			Expect(err).ToNot(HaveOccurred())
			conn.EXPECT().PublishRaw("choria.lifecycle.event.startup.ginkgo", []byte(`{"protocol":"io.choria.lifecycle.v1.startup","id":"01e72410-d734-4611-9485-8c6a2dd2579b","identity":"ginkgo.example.net","component":"ginkgo","timestamp":1535106973,"version":"1.2.3"}`))
			err = PublishEvent(event, conn)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support cloud events format", func() {
			mockTime = 1535106973
			mockID = "01e72410-d734-4611-9485-8c6a2dd2579b"

			event, err := New(Startup, Component("ginkgo"), Version("1.2.3"), Identity("ginkgo.example.net"))
			Expect(err).ToNot(HaveOccurred())
			event.SetFormat(CloudEventV1Format)
			conn.EXPECT().PublishRaw("choria.lifecycle.event.startup.ginkgo", []byte(`{"data":{"protocol":"io.choria.lifecycle.v1.startup","id":"01e72410-d734-4611-9485-8c6a2dd2579b","identity":"ginkgo.example.net","component":"ginkgo","timestamp":1535106973,"version":"1.2.3"},"id":"01e72410-d734-4611-9485-8c6a2dd2579b","source":"io.choria.lifecycle","specversion":"1.0","subject":"ginkgo","time":"2018-08-24T10:36:13Z","type":"startup"}`))
			err = PublishEvent(event, conn)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
