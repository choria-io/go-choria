package tally

import (
	"io/ioutil"
	"time"

	"github.com/choria-io/go-lifecycle"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"testing"
)

func TestChoria(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tally")
}

var _ = Describe("Tally", func() {
	var (
		logger   = logrus.NewEntry(logrus.New())
		recorder *Recorder
	)

	BeforeEach(func() {
		logger.Logger.SetOutput(ioutil.Discard)
		registerStats = false
		recorder = &Recorder{
			observed: make(map[uint64]*observation),
			options: &options{
				Component:  "ginkgo",
				StatPrefix: "tally",
				Log:        logger,
			},
		}
		recorder.createStats()
	})

	Describe("maintenance", func() {
		It("Should not delete current nodes", func() {
			event, err := lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
			Expect(err).ToNot(HaveOccurred())

			recorder.process(event)
			Expect(recorder.observed).To(HaveLen(1))
			recorder.maintenance()
			Expect(recorder.observed).To(HaveLen(1))
		})

		It("Should not delete old nodes", func() {
			event, err := lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
			Expect(err).ToNot(HaveOccurred())

			recorder.process(event)
			Expect(recorder.observed).To(HaveLen(1))

			h := hostHash("ginkgo.example.net")
			recorder.observed[h].ts = recorder.observed[h].ts.Add(-1 * (time.Hour + 2*time.Minute))
			recorder.maintenance()
			Expect(recorder.observed).To(HaveLen(0))
		})
	})

	Describe("process", func() {
		Describe("Shutdown Events", func() {
			It("Should handle existing nodes", func() {
				event, err := lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				recorder.processStartup(event)
				Expect(recorder.observed).To(HaveLen(1))

				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(1.0))

				event, err = lifecycle.New(lifecycle.Shutdown, lifecycle.Component("ginkgo"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())
				recorder.processShutdown(event)

				Expect(recorder.observed).To(HaveLen(0))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(0.0))
			})

			It("Should handle new nodes", func() {
				event, err := lifecycle.New(lifecycle.Shutdown, lifecycle.Component("ginkgo"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())
				recorder.processShutdown(event)

				Expect(recorder.observed).To(HaveLen(0))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(0.0))
			})
		})

		Describe("Startup Events", func() {
			It("Should handle new nodes", func() {
				event, err := lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				Expect(recorder.observed).To(HaveLen(0))
				recorder.processStartup(event)
				Expect(recorder.observed).To(HaveLen(1))

				h := hostHash("ginkgo.example.net")
				Expect(recorder.observed[h].version).To(Equal("1.2.3"))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(1.0))
			})

			It("Should handle existing nodes", func() {
				event, err := lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				Expect(recorder.observed).To(HaveLen(0))
				recorder.processStartup(event)
				Expect(recorder.observed).To(HaveLen(1))

				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(1.0))

				event, err = lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.4"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				recorder.processStartup(event)

				h := hostHash("ginkgo.example.net")
				Expect(recorder.observed[h].version).To(Equal("1.2.4"))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(0.0))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.4")).To(Equal(1.0))
			})
		})

		Describe("Alive Events", func() {
			It("Should handle new hosts", func() {
				event, err := lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				Expect(recorder.observed).To(HaveLen(0))
				recorder.processAlive(event)
				Expect(recorder.observed).To(HaveLen(1))

				h := hostHash("ginkgo.example.net")
				Expect(recorder.observed[h].version).To(Equal("1.2.3"))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(1.0))
			})

			It("Should handle old hosts", func() {
				event, err := lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				recorder.processAlive(event)
				Expect(recorder.observed).To(HaveLen(1))

				h := hostHash("ginkgo.example.net")
				Expect(recorder.observed[h].version).To(Equal("1.2.3"))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(1.0))

				recorder.processAlive(event)

				Expect(recorder.observed).To(HaveLen(1))
				Expect(recorder.observed[h].version).To(Equal("1.2.3"))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(1.0))
			})

			It("Should handle updated hosts", func() {
				event, err := lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				h := hostHash("ginkgo.example.net")

				recorder.processAlive(event)
				Expect(recorder.observed[h].version).To(Equal("1.2.3"))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.3")).To(Equal(1.0))

				event, err = lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.4"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				recorder.processAlive(event)
				Expect(recorder.observed[h].version).To(Equal("1.2.4"))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.2")).To(Equal(0.0))
				Expect(getPromValue(recorder.eventsTally, "ginkgo", "1.2.4")).To(Equal(1.0))
			})
		})
	})
})

func getPromValue(ctr *prometheus.GaugeVec, labels ...string) float64 {
	pb := &dto.Metric{}
	m, err := ctr.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0
	}

	if m.Write(pb) != nil {
		return 0
	}

	return pb.GetGauge().GetValue()
}
