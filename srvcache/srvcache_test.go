package srvcache

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSRVCache(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "SRVCache")
}

var _ = Describe("LookupSRV", func() {
	var ctr = 1
	var resolver = func(service string, proto string, name string) (cname string, addrs []*net.SRV, err error) {
		a := &net.SRV{
			Target:   fmt.Sprintf("1.2.3.%d", ctr),
			Priority: 1,
			Weight:   1,
			Port:     5222,
		}

		ctr++

		return "test.example.net", []*net.SRV{a}, nil
	}

	It("Should do a lookup and cache it and serve from the cache", func() {
		ctr = 1

		cname, addrs, err := LookupSRV("", "", "test.example.net", resolver)
		Expect(cname).To(Equal("test.example.net"))
		Expect(addrs[0].Target).To(Equal("1.2.3.1"))
		Expect(err).ToNot(HaveOccurred())

		cname, addrs, err = LookupSRV("", "", "test.example.net", resolver)
		Expect(cname).To(Equal("test.example.net"))
		Expect(addrs[0].Target).To(Equal("1.2.3.1"))
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should age entries out of the cache", func() {
		ctr = 1

		_, addrs, _ := LookupSRV("", "", "another.example.net", resolver)
		Expect(addrs[0].Target).To(Equal("1.2.3.1"))
		_, addrs, _ = LookupSRV("", "", "another.example.net", resolver)
		Expect(addrs[0].Target).To(Equal("1.2.3.1"))

		_, found := cache[query{"", "", "another.example.net"}]
		Expect(found).To(BeTrue())

		maxage = time.Duration(-1 * time.Second)

		_, addrs = retrieve(query{"", "", "another.example.net"})
		Expect(addrs).To(BeNil())

		_, found = cache[query{"", "", "another.example.net"}]
		Expect(found).To(BeFalse())
	})
})
