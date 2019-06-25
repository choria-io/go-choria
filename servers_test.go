package srvcache

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestChoria(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Choria SRV Cache")
}

var _ = Describe("Servers", func() {
	var s *servers

	BeforeEach(func() {
		s = &servers{servers: []Server{NewServer("h1", 8080, "http"), NewServer("h2", 8080, "https")}}
	})

	Describe("Count", func() {
		It("Should store the supplied servers", func() {
			Expect(s.servers).To(HaveLen(2))
			Expect(s.Count()).To(Equal(2))
		})
	})

	Describe("Servers", func() {
		It("Should return the stored servers", func() {
			srvs := s.Servers()
			Expect(srvs).To(HaveLen(2))
			Expect(srvs[0].Host()).To(Equal("h1"))
			Expect(srvs[1].Host()).To(Equal("h2"))
		})
	})

	Describe("Each", func() {
		It("Should yield all instances and allow edits", func() {
			s.Each(func(srv Server) {
				srv.SetHost("test" + srv.Host())
			})

			hps := s.HostPorts()
			Expect(hps).To(Equal([]string{
				"testh1:8080",
				"testh2:8080",
			}))
		})
	})

	Describe("Strings", func() {
		It("Should return the right strings", func() {
			Expect(s.Strings()).To(Equal([]string{
				"http://h1:8080",
				"https://h2:8080",
			}))
		})
	})

	Describe("URLs", func() {
		It("Should produce correct URLs", func() {
			urls, err := s.URLs()
			Expect(err).ToNot(HaveOccurred())
			Expect(urls[0].Host).To(Equal("h1:8080"))
			Expect(urls[0].Scheme).To(Equal("http"))
			Expect(urls[1].Host).To(Equal("h2:8080"))
			Expect(urls[1].Scheme).To(Equal("https"))
		})
	})

	Describe("HostPorts", func() {
		It("Should produce correct hps", func() {
			Expect(s.HostPorts()).To(Equal([]string{
				"h1:8080",
				"h2:8080",
			}))
		})
	})
})
