// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package broker_mappings

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/integration/testbroker"
	"github.com/choria-io/go-choria/integration/testutil"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
)

func TestBrokerRemapping(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration/Broker Remapping")
}

var _ = Describe("Authentication", func() {
	var (
		ctx     context.Context
		cancel  context.CancelFunc
		wg      sync.WaitGroup
		logger  *logrus.Logger
		logbuff *gbytes.Buffer
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 45*time.Second)
		DeferCleanup(func() {
			cancel()
			Eventually(logbuff, 5).Should(gbytes.Say("Choria Network Broker shut down"))
		})

		logbuff, logger = testutil.GbytesLogger(logrus.DebugLevel)
	})

	Describe("Mappings", func() {
		BeforeEach(func() {
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/mappings.conf", logger)
			Expect(err).ToNot(HaveOccurred())

			Eventually(logbuff, 2).Should(gbytes.Say("TLS required for client connections"))
			Eventually(logbuff, 1).Should(gbytes.Say("Server is ready"))
		})

		It("Should add correct mappings", func() {
			nc, err := nats.Connect("tls://localhost:4222",
				nats.ClientCert(testutil.CertPath("one", "rip.mcollective"), testutil.KeyPath("one", "rip.mcollective")),
				nats.RootCAs(testutil.CertPath("one", "ca")),
			)
			Expect(err).ToNot(HaveOccurred())
			defer nc.Close()

			Eventually(logbuff, 1).Should(gbytes.Say("Registering user '' in account 'choria'"))
			Expect(nc.ConnectedUrl()).To(Equal("tls://localhost:4222"))

			sub, err := nc.SubscribeSync("registration.>")
			Expect(err).ToNot(HaveOccurred())

			Expect(nc.Publish("in.registration.dev1.example.net", []byte("dev1.example.net"))).To(Succeed())
			Expect(nc.Publish("in.registration.dev2.example.net", []byte("dev2.example.net"))).To(Succeed())
			Expect(nc.Publish("in.registration.dev3.example.net", []byte("dev3.example.net"))).To(Succeed())
			Expect(nc.Publish("in.registration.dev4.example.net", []byte("dev4.example.net"))).To(Succeed())
			Expect(nc.Publish("in.registration.dev5.example.net", []byte("dev5.example.net"))).To(Succeed())
			Expect(nc.Publish("in.registration.dev1.example.net", []byte("dev1.example.net"))).To(Succeed())

			check := func(body, subj string) {
				msg, err := sub.NextMsg(time.Second)
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
				ExpectWithOffset(1, msg.Data).To(Equal([]byte(body)))
				ExpectWithOffset(1, msg.Subject).To(Equal(subj))
			}

			check("dev1.example.net", "registration.2.dev1.example.net")
			check("dev2.example.net", "registration.1.dev2.example.net")
			check("dev3.example.net", "registration.0.dev3.example.net")
			check("dev4.example.net", "registration.2.dev4.example.net")
			check("dev5.example.net", "registration.1.dev5.example.net")
			check("dev1.example.net", "registration.2.dev1.example.net")
		})
	})
})
