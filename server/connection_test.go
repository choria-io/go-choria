package server

import (
	"context"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Server/Connection", func() {
	var _ = Describe("brokerUrls", func() {
		var (
			cfg    *config.Config
			fw     *choria.Framework
			srv    *Instance
			err    error
			ctx    context.Context
			cancel func()
		)

		BeforeEach(func() {
			cfg = config.NewConfigForTests()
			Expect(err).ToNot(HaveOccurred())

			cfg.DisableTLS = true
			cfg.Choria.MiddlewareHosts = []string{"d1:4222", "d2:4222"}

			fw, err = choria.NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			srv, err = NewInstance(fw)
			Expect(err).ToNot(HaveOccurred())

			logrus.SetLevel(logrus.FatalLevel)

			ctx, cancel = context.WithCancel(context.Background())
		})

		AfterEach(func() {
			cancel()
		})

		It("Should support provisioning", func() {
			build.ProvisionModeDefault = "true"
			build.ProvisionBrokerURLs = "nats1:4222, nats2:4222"
			cfg.InitiatedByServer = true

			servers, err := srv.brokerUrls(ctx)
			Expect(err).ToNot(HaveOccurred())

			found := servers.Servers()
			Expect(found[0].Host()).To(Equal("nats1"))
			Expect(found[1].Host()).To(Equal("nats2"))
		})

		It("Should fail gracefully for incorrect format provisioning servers", func() {
			build.ProvisionModeDefault = "true"
			build.ProvisionBrokerURLs = "invalid stuff"

			servers, err := srv.brokerUrls(ctx)
			Expect(err).ToNot(HaveOccurred())

			found := servers.Servers()
			Expect(found).To(HaveLen(2))
			Expect(found[0].Host()).To(Equal("d1"))
			Expect(found[1].Host()).To(Equal("d2"))
		})

		It("Should fail gracefully when no servers are compiled in but provisioning is on", func() {
			build.ProvisionModeDefault = "true"
			build.ProvisionBrokerURLs = ""

			servers, err := srv.brokerUrls(ctx)
			Expect(err).ToNot(HaveOccurred())

			found := servers.Servers()
			Expect(found).To(HaveLen(2))
			Expect(found[0].Host()).To(Equal("d1"))
			Expect(found[1].Host()).To(Equal("d2"))
		})

		It("Should default to unprovisioned mode", func() {
			servers, err := srv.brokerUrls(ctx)
			Expect(err).ToNot(HaveOccurred())

			found := servers.Servers()
			Expect(found).To(HaveLen(2))
			Expect(found[0].Host()).To(Equal("d1"))
			Expect(found[1].Host()).To(Equal("d2"))
		})
	})
})
