package federation

import (
	"context"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Pooled Worker", func() {
	var s, l, r *pooledWorker

	BeforeEach(func() {
		log.SetOutput(io.Discard)
		logger := log.WithFields(log.Fields{"test": "true"})
		broker := &FederationBroker{}

		s, _ = PooledWorkerFactory("socket", 1, Unconnected, 1000, broker, logger, func(ctx context.Context, w *pooledWorker, i int, l *log.Entry) {})
		l, _ = PooledWorkerFactory("left", 1, Unconnected, 1000, broker, logger, func(ctx context.Context, w *pooledWorker, i int, l *log.Entry) {})
		r, _ = PooledWorkerFactory("right", 1, Unconnected, 1000, broker, logger, func(ctx context.Context, w *pooledWorker, i int, l *log.Entry) {})
	}, 10)

	It("Should correctly initialize", func() {
		Expect(s.Name()).To(Equal("socket"))
		Expect(s.mode).To(Equal(Unconnected))
		Expect(s.capacity).To(Equal(1000))
		Expect(s.Input()).To(HaveCap(1000))
		Expect(s.Output()).To(HaveCap(1000))
		Expect(s.Ready()).To(BeTrue())
	})

	It("Should correctly plug chainables into each other", func() {
		Expect(l.Output()).ToNot(Equal(s.Input()))
		err := s.From(l)
		Expect(err).ToNot(HaveOccurred())
		Expect(l.Output()).To(Equal(s.Input()))

		Expect(r.Input()).ToNot(Equal(s.Output()))
		err = s.To(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(r.Input()).To(Equal(s.Output()))
	})
})
