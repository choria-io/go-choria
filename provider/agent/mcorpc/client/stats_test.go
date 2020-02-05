package client

import (
	"sort"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("McoRPC/Client/Stats", func() {
	Describe("All", func() {
		var (
			s *Stats
		)

		BeforeEach(func() {
			s = NewStats()
		})

		Describe("Merge", func() {
			It("Should correctly merge", func() {
				other := NewStats()

				s.SetDiscoveredNodes([]string{"host1", "host2", "host3"})
				other.SetDiscoveredNodes([]string{"host2", "host3"})

				other.StartPublish()
				other.publishStart = other.publishStart.Add(-10 * time.Second)

				other.RecordReceived("host2")
				other.RecordReceived("host3")
				other.RecordReceived("host4")
				other.PassedRequestInc()
				other.FailedRequestInc()

				other.EndPublish()

				s.Merge(other)

				Expect(s.UnexpectedResponseFrom()).To(Equal([]string{"host4"}))
				Expect(s.outstandingNodes.Hosts()).To(Equal([]string{"host1"}))
				Expect(s.FailCount()).To(Equal(1))
				Expect(s.OKCount()).To(Equal(1))
				Expect(s.PublishDuration()).To(BeNumerically("~", 10*time.Second, 10*time.Millisecond))
			})
		})

		Describe("SetAgent / Agent", func() {
			It("Should set and get the right agent", func() {
				Expect(s.Agent()).To(Equal(""))
				s.SetAgent("foo")
				Expect(s.Agent()).To(Equal("foo"))
			})
		})

		Describe("SetAction / Action", func() {
			It("Should set and get the right action", func() {
				Expect(s.Action()).To(Equal(""))
				s.SetAction("foo")
				Expect(s.Action()).To(Equal("foo"))
			})
		})

		Describe("All", func() {
			It("Should correctly determine if all nodes have completed", func() {
				s.SetDiscoveredNodes([]string{"host1", "host2"})
				Expect(s.All()).To(BeFalse())
				s.RecordReceived("host1")
				Expect(s.All()).To(BeFalse())
				s.RecordReceived("host2")
				Expect(s.All()).To(BeTrue())
			})
		})

		Describe("NoResponseFrom", func() {
			It("Should return the correct list", func() {
				s.SetDiscoveredNodes([]string{"host1", "host2"})

				nr := s.NoResponseFrom()
				sort.Strings(nr)
				Expect(nr).To(Equal(strings.Fields("host1 host2")))

				s.RecordReceived("host1")
				Expect(s.NoResponseFrom()).To(Equal([]string{"host2"}))

				s.RecordReceived("host2")
				Expect(s.NoResponseFrom()).To(Equal([]string{}))
			})
		})

		Describe("UnexpectedResponseFrom", func() {
			It("Should report correctly", func() {
				s.SetDiscoveredNodes([]string{"host1", "host2"})
				Expect(s.UnexpectedResponseFrom()).To(Equal([]string{}))
				s.RecordReceived("host3")
				Expect(s.UnexpectedResponseFrom()).To(Equal([]string{"host3"}))
			})
		})

		Describe("WaitingFor", func() {
			It("Should report nodes correctly", func() {
				s.SetDiscoveredNodes([]string{"host1", "host2"})
				Expect(s.outstandingNodes.Count()).To(Equal(2))
				Expect(s.outstandingNodes.HaveAny("host1", "host2")).To(BeTrue())
				Expect(s.WaitingFor([]string{"host1", "host2"})).To(BeTrue())
				Expect(s.WaitingFor([]string{"host3", "host4"})).To(BeFalse())
			})
		})

		Describe("SetDiscoveredNodes", func() {
			It("Should set the node and outstanding nodes", func() {
				s.outstandingNodes.AddHosts("host100")
				s.SetDiscoveredNodes([]string{"host1", "host2"})
				Expect(s.discoveredNodes).To(Equal([]string{"host1", "host2"}))

				o := s.outstandingNodes.Hosts()
				sort.Strings(o)
				Expect(o).To(Equal([]string{"host1", "host2"}))
			})
		})

		Describe("FailedRequestInc", func() {
			It("Should increase the count", func() {
				Expect(s.failed.Load()).To(Equal(int32(0)))
				s.FailedRequestInc()
				Expect(s.failed.Load()).To(Equal(int32(1)))
			})
		})

		Describe("PassedRequestInc", func() {
			It("Should increase the count", func() {
				Expect(s.passed.Load()).To(Equal(int32(0)))
				s.PassedRequestInc()
				Expect(s.passed.Load()).To(Equal(int32(1)))
			})
		})

		Describe("RecordReceived", func() {
			It("Should handle outstanding nodes", func() {
				s.SetDiscoveredNodes([]string{"host1", "host2"})
				Expect(s.responses.Load()).To(Equal(int32(0)))
				s.RecordReceived("host2")
				Expect(s.responses.Load()).To(Equal(int32(1)))
				Expect(s.NoResponseFrom()).To(Equal([]string{"host1"}))
			})

			It("Should handle unexpected nodes", func() {
				s.SetDiscoveredNodes([]string{"host1", "host2"})
				Expect(s.responses.Load()).To(Equal(int32(0)))
				s.RecordReceived("host3")
				Expect(s.responses.Load()).To(Equal(int32(1)))
				Expect(s.UnexpectedResponseFrom()).To(Equal([]string{"host3"}))
			})
		})

		Describe("DiscoveredCount", func() {
			It("Should return the right length", func() {
				Expect(s.DiscoveredCount()).To(Equal(0))
				s.SetDiscoveredNodes([]string{"host1", "host2"})
				Expect(s.DiscoveredCount()).To(Equal(2))
			})
		})

		Describe("FailCount", func() {
			It("Should return the right length", func() {
				Expect(s.FailCount()).To(Equal(0))
				s.FailedRequestInc()
				Expect(s.FailCount()).To(Equal(1))
			})
		})

		Describe("OKCount", func() {
			It("Should return the right length", func() {
				Expect(s.OKCount()).To(Equal(0))
				s.PassedRequestInc()
				Expect(s.OKCount()).To(Equal(1))
			})
		})

		Describe("ResponsesCount", func() {
			It("Should return the right length", func() {
				Expect(s.ResponsesCount()).To(Equal(0))
				s.RecordReceived("host1")
				Expect(s.ResponsesCount()).To(Equal(1))
			})
		})

		Describe("StartPublish", func() {
			It("Should start the clock if it has not yet started", func() {
				Expect(s.publishStart).To(BeZero())
				s.StartPublish()

				t := s.publishStart
				Expect(t).ToNot(BeZero())

				s.StartPublish()
				Expect(s.publishStart).To(Equal(t))
			})
		})

		Describe("EndPublish", func() {
			It("Should end the publishing if its not already ended", func() {
				Expect(s.publishEnd).To(BeZero())
				s.StartPublish()
				s.EndPublish()

				t := s.publishEnd
				Expect(t).ToNot(BeZero())

				s.EndPublish()
				Expect(s.publishEnd).To(Equal(t))
			})
		})

		Describe("PublishDuration", func() {
			It("Should handle in-progress or not yet started requests", func() {
				_, err := s.PublishDuration()
				Expect(err).To(MatchError("publishing is not completed"))

				s.StartPublish()
				_, err = s.PublishDuration()
				Expect(err).To(MatchError("publishing is not completed"))

				s.publishStart = time.Now().Add(-10 * time.Second)

				Expect(s.publishStart).ToNot(BeZero())
				Expect(s.publishing).To(BeTrue())
				Expect(s.publishEnd).To(BeZero())

				s.EndPublish()

				Expect(s.publishEnd).ToNot(BeZero())
				Expect(s.publishing).To(BeFalse())

				_, err = s.PublishDuration()
				Expect(err).ToNot(HaveOccurred())
			})

			It("Should determine the correct duration", func() {
				t := time.Now()

				s.StartPublish()
				s.publishStart = t.Add(-10 * time.Second)
				s.EndPublish()

				Expect(s.PublishDuration()).To(BeNumerically("~", 10*time.Second, 100*time.Millisecond))
			})

			It("Should support multiple publishes", func() {
				t := time.Now()

				s.StartPublish()
				s.publishStart = t.Add(-10 * time.Second)
				s.EndPublish()

				t = time.Now()
				s.StartPublish()
				s.publishStart = t.Add(-10 * time.Second)
				s.EndPublish()

				Expect(s.PublishDuration()).To(BeNumerically("~", 20*time.Second, 100*time.Millisecond))
			})
		})

		Describe("Start", func() {
			It("Should start the clock if its not already started", func() {
				Expect(s.start).To(BeZero())
				s.Start()
				t := s.start
				s.Start()
				Expect(s.start).To(Equal(t))
			})
		})

		Describe("End", func() {
			It("Should end the clock if its not already end", func() {
				Expect(s.end).To(BeZero())
				s.End()
				t := s.end
				s.End()
				Expect(s.end).To(Equal(t))
			})
		})

		Describe("RequestDuration", func() {
			It("Should detect unfinished requests", func() {
				s.Start()
				_, err := s.RequestDuration()
				Expect(err).To(MatchError("request is not completed"))
			})

			It("Should handle completed requests", func() {
				s.Start()
				s.start = time.Now().Add(-1 * time.Second)
				s.End()
				d, err := s.RequestDuration()
				Expect(err).ToNot(HaveOccurred())
				Expect(d).To(BeNumerically("~", time.Second, 50*time.Millisecond))
			})
		})
	})
})
