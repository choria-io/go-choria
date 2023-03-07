// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tally

import (
	"io"
	"testing"
	"time"

	"github.com/choria-io/go-choria/lifecycle"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestChoria(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lifecycle/Tally")
}

var _ = Describe("Tally", func() {
	var (
		logger   = logrus.NewEntry(logrus.New())
		recorder *Recorder
	)

	BeforeEach(func() {
		logger.Logger.SetOutput(io.Discard)
		registerStats = false
		recorder = &Recorder{
			active:   1,
			observed: make(map[string]*observations),
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
			Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))
			recorder.maintenance()
			Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))
		})

		It("Should delete old nodes", func() {
			event, err := lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
			Expect(err).ToNot(HaveOccurred())

			recorder.process(event)
			Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))

			recorder.observed["ginkgo"].hosts["ginkgo.example.net"].ts = recorder.observed["ginkgo"].hosts["ginkgo.example.net"].ts.Add(-1 * (60 * time.Minute))
			recorder.maintenance()
			Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))

			recorder.observed["ginkgo"].hosts["ginkgo.example.net"].ts = recorder.observed["ginkgo"].hosts["ginkgo.example.net"].ts.Add(-1 * (90 * time.Minute))
			recorder.maintenance()
			Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(0))
		})
	})

	Describe("elections", func() {
		It("Should correctly label metrics", func() {
			event, err := lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
			Expect(err).ToNot(HaveOccurred())

			recorder.process(event)
			Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))

			Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))

			recorder.lostCb()

			event, err = lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.4"), lifecycle.Identity("other.example.net"))
			Expect(err).ToNot(HaveOccurred())

			recorder.process(event)
			Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(2))

			Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))
			Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.4", "0")).To(Equal(1.0))

			recorder.wonCb()
			event, err = lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.4"), lifecycle.Identity("foo.example.net"))
			Expect(err).ToNot(HaveOccurred())

			recorder.process(event)
			Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(3))

			Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))
			Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.4", "0")).To(Equal(1.0))
			Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.4", "1")).To(Equal(1.0))
		})
	})

	Describe("process", func() {
		Describe("Shutdown Events", func() {
			It("Should handle existing nodes", func() {
				event, err := lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				recorder.process(event)
				Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))

				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))

				event, err = lifecycle.New(lifecycle.Shutdown, lifecycle.Component("ginkgo"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())
				recorder.process(event)

				Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(0))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3")).To(Equal(0.0))
			})

			It("Should handle new nodes", func() {
				event, err := lifecycle.New(lifecycle.Shutdown, lifecycle.Component("ginkgo"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())
				recorder.process(event)

				Expect(recorder.observed).To(HaveLen(0))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3")).To(Equal(0.0))
			})
		})

		Describe("Startup Events", func() {
			It("Should handle new nodes", func() {
				event, err := lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				Expect(recorder.observed).To(HaveLen(0))
				recorder.process(event)
				Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))

				Expect(recorder.observed["ginkgo"].hosts["ginkgo.example.net"].version).To(Equal("1.2.3"))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))
			})

			It("Should handle existing nodes", func() {
				event, err := lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				Expect(recorder.observed).To(HaveLen(0))
				recorder.process(event)
				Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))

				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))

				event, err = lifecycle.New(lifecycle.Startup, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.4"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				recorder.process(event)

				Expect(recorder.observed["ginkgo"].hosts["ginkgo.example.net"].version).To(Equal("1.2.4"))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(0.0))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.4", "1")).To(Equal(1.0))
			})
		})

		Describe("Governor Events", func() {
			It("Should handle governor events", func() {
				event, err := lifecycle.New(lifecycle.Governor, lifecycle.Component("ginkgo"), lifecycle.GovernorName("GINKGO"), lifecycle.GovernorType(lifecycle.GovernorEnterEvent))
				Expect(err).ToNot(HaveOccurred())
				recorder.process(event)
				Expect(getPromCountValue(recorder.governorEvents, "ginkgo", "GINKGO", "enter", "1")).To(Equal(1.0))
				Expect(getPromCountValue(recorder.governorEvents, "ginkgo", "GINKGO", "exit", "1")).To(Equal(0.0))

				event, err = lifecycle.New(lifecycle.Governor, lifecycle.Component("ginkgo"), lifecycle.GovernorName("GINKGO"), lifecycle.GovernorType(lifecycle.GovernorExitEvent))
				Expect(err).ToNot(HaveOccurred())
				recorder.process(event)
				Expect(getPromCountValue(recorder.governorEvents, "ginkgo", "GINKGO", "enter", "1")).To(Equal(1.0))
				Expect(getPromCountValue(recorder.governorEvents, "ginkgo", "GINKGO", "exit", "1")).To(Equal(1.0))
			})
		})

		Describe("Alive Events", func() {
			It("Should handle new hosts", func() {
				event, err := lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				Expect(recorder.observed).To(HaveLen(0))
				recorder.process(event)
				Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))

				Expect(recorder.observed["ginkgo"].hosts["ginkgo.example.net"].version).To(Equal("1.2.3"))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))
			})

			It("Should handle old hosts", func() {
				event, err := lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				recorder.process(event)
				Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))
				Expect(recorder.observed["ginkgo"].hosts["ginkgo.example.net"].version).To(Equal("1.2.3"))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))

				recorder.observed["ginkgo"].hosts["ginkgo.example.net"].ts = time.Now().Add(-120 * time.Minute)

				recorder.process(event)

				Expect(recorder.observed["ginkgo"].hosts).To(HaveLen(1))
				Expect(recorder.observed["ginkgo"].hosts["ginkgo.example.net"].version).To(Equal("1.2.3"))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))
				Expect(recorder.observed["ginkgo"].hosts["ginkgo.example.net"].ts).To(BeTemporally("~", time.Now(), time.Second))
			})

			It("Should handle updated hosts", func() {
				event, err := lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.3"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				recorder.process(event)
				Expect(recorder.observed["ginkgo"].hosts["ginkgo.example.net"].version).To(Equal("1.2.3"))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.3", "1")).To(Equal(1.0))

				event, err = lifecycle.New(lifecycle.Alive, lifecycle.Component("ginkgo"), lifecycle.Version("1.2.4"), lifecycle.Identity("ginkgo.example.net"))
				Expect(err).ToNot(HaveOccurred())

				recorder.process(event)
				Expect(recorder.observed["ginkgo"].hosts["ginkgo.example.net"].version).To(Equal("1.2.4"))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.2", "1")).To(Equal(0.0))
				Expect(getPromGaugeValue(recorder.versionsTally, "ginkgo", "1.2.4", "1")).To(Equal(1.0))
			})
		})
	})
})

func getPromCountValue(ctr *prometheus.CounterVec, labels ...string) float64 {
	pb := &dto.Metric{}
	m, err := ctr.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0
	}

	if m.Write(pb) != nil {
		return 0
	}

	return pb.GetCounter().GetValue()
}

func getPromGaugeValue(ctr *prometheus.GaugeVec, labels ...string) float64 {
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
