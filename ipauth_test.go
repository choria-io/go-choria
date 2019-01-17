package network

import (
	"io/ioutil"
	"net"

	"github.com/nats-io/gnatsd/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Network Broker/IPAuth", func() {
	var (
		log  *logrus.Entry
		auth *IPAuth
		user *server.User
	)

	BeforeEach(func() {
		logger := logrus.New()
		logger.Out = ioutil.Discard
		log = logrus.NewEntry(logger)
		auth = &IPAuth{
			allowList: []string{},
			log:       log,
		}
		user = &server.User{
			Username:    "bob",
			Password:    "secret",
			Permissions: &server.Permissions{},
		}
	})

	Describe("remoteInClientAllowList", func() {
		It("Should allow all when no allowlist is set", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			Expect(auth.remoteInClientAllowList(&net.IPAddr{IP: ipv4Addr})).To(BeTrue())
		})

		It("Should handle nil remotes", func() {
			Expect(auth.remoteInClientAllowList(nil)).To(BeTrue())
		})

		It("Should handle invalid remotes", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.allowList = []string{"192.0.2.1/24"}
			Expect(auth.remoteInClientAllowList(&net.IPAddr{IP: ipv4Addr})).To(BeFalse())
		})

		It("Should handle simple strings", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.allowList = []string{"192.0.2.1"}
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeTrue())
		})

		It("Should handle subnets", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.allowList = []string{"192.0.0.0/8"}
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeTrue())
		})

		It("Should support IPv6", func() {
			auth.allowList = []string{
				"2a00:1450::/32",
				"2a01:1450:4002:801::200e",
			}

			ipv6Addr, _, err := net.ParseCIDR("2a00:1450:4002:801::200e/64")
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv6Addr, Port: 1232})).To(BeTrue())

			ipv6Addr, _, err = net.ParseCIDR("2a01:1450:4002:801::200e/64")
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv6Addr, Port: 1232})).To(BeTrue())

			ipv6Addr, _, err = net.ParseCIDR("2a02:1450:4002:801::200e/64")
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv6Addr, Port: 1232})).To(BeFalse())
		})

		It("Should be false for un matched nodes", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.allowList = []string{"127.0.0.0/8"}
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeFalse())

			ipv4Addr, _, err = net.ParseCIDR("127.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeTrue())
		})
	})

	Describe("setServerPermissions", func() {
		It("Should set correct permissions", func() {
			auth.setServerPermissions(user)

			Expect(user.Permissions.Publish.Allow).To(Equal([]string{
				"*.broadcast.agent.registration",
			}))

			Expect(user.Permissions.Publish.Deny).To(Equal([]string{
				"*.broadcast.agent.>",
				"*.node.>",
				"choria.federation.*.federation",
			}))

			Expect(user.Permissions.Subscribe.Allow).To(HaveLen(0))
			Expect(user.Permissions.Subscribe.Deny).To(Equal([]string{
				"*.reply.>",
				"choria.federation.>",
				"choria.lifecycle.>",
			}))
		})
	})
})
