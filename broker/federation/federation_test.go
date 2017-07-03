package federation

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/choria-io/go-choria/mcollective"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func TestFederation(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Federation Suite")
}

var _ = Describe("Federation Broker", func() {
	It("Should initialize correctly", func() {
		log.SetOutput(ioutil.Discard)

		choria, err := mcollective.New("testdata/federation.cfg")
		Expect(err).ToNot(HaveOccurred())

		fb, err := NewFederationBroker("test_cluster", "test_instance", choria)
		Expect(err).ToNot(HaveOccurred())

		Expect(fb.Stats.Status).To(Equal("unknown"))
		Expect(fb.Stats.CollectiveStats.ConnectedServer).To(Equal("unknown"))
		Expect(fb.Stats.FederationStats.ConnectedServer).To(Equal("unknown"))
		Expect(fb.Stats.StartTime).To(BeTemporally("~", time.Now()))
	})
})
