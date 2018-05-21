package server

import (
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/srvcache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Server/Connection", func() {
	var _ = Describe("brokerUrls", func() {
		var (
			cfg *config.Config
			fw  *choria.Framework
			srv *Instance
			err error
		)

		BeforeEach(func() {
			cfg, err = config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			cfg.DisableTLS = true
			cfg.Choria.MiddlewareHosts = []string{"d1:4222", "d2:4222"}

			fw, err = choria.NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			srv, err = NewInstance(fw)
			Expect(err).ToNot(HaveOccurred())

			logrus.SetLevel(logrus.FatalLevel)
		})

		It("Should support provisioning", func() {
			build.ProvisionModeDefault = "true"
			build.ProvisionBrokerURLs = "nats1:4222, nats2:4222"

			servers, err := srv.brokerUrls()
			Expect(err).ToNot(HaveOccurred())

			expected := []srvcache.Server{
				srvcache.Server{Host: "nats1", Port: 4222, Scheme: "nats"},
				srvcache.Server{Host: "nats2", Port: 4222, Scheme: "nats"},
			}

			Expect(servers).To(Equal(expected))
		})

		It("Should fail gracefully for incorrect format provisioning servers", func() {
			build.ProvisionModeDefault = "true"
			build.ProvisionBrokerURLs = "invalid stuff"

			servers, err := srv.brokerUrls()
			Expect(err).ToNot(HaveOccurred())

			expected := []srvcache.Server{
				srvcache.Server{Host: "d1", Port: 4222, Scheme: "nats"},
				srvcache.Server{Host: "d2", Port: 4222, Scheme: "nats"},
			}

			Expect(servers).To(Equal(expected))
		})

		It("Should fail gracefully when no servers are compiled in but provisioning is on", func() {
			build.ProvisionModeDefault = "true"
			build.ProvisionBrokerURLs = ""

			servers, err := srv.brokerUrls()
			Expect(err).ToNot(HaveOccurred())

			expected := []srvcache.Server{
				srvcache.Server{Host: "d1", Port: 4222, Scheme: "nats"},
				srvcache.Server{Host: "d2", Port: 4222, Scheme: "nats"},
			}

			Expect(servers).To(Equal(expected))
		})

		It("Should default to unprovisioned mode", func() {
			servers, err := srv.brokerUrls()
			Expect(err).ToNot(HaveOccurred())

			expected := []srvcache.Server{
				srvcache.Server{Host: "d1", Port: 4222, Scheme: "nats"},
				srvcache.Server{Host: "d2", Port: 4222, Scheme: "nats"},
			}

			Expect(servers).To(Equal(expected))
		})
	})
})
