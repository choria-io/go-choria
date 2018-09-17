package provtarget

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/provtarget/builddefaults"
	"github.com/choria-io/go-choria/srvcache"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestServer(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provision")
}

var _ = Describe("Provision", func() {
	var (
		mockctl      *gomock.Controller
		mockresolver *MockTargetResolver
		log          *logrus.Entry
		ctx          context.Context
		cancel       func()
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockresolver = NewMockTargetResolver(mockctl)
		mockresolver.EXPECT().Name().Return("Mock Resolver").AnyTimes()
		RegisterTargetResolver(builddefaults.Provider())
		ctx, cancel = context.WithCancel(context.Background())
		log = logrus.NewEntry(logrus.New())
		log.Logger.Out = ioutil.Discard
	})

	AfterEach(func() {
		mockctl.Finish()
		cancel()
	})

	Describe("RegisterTargetResolver", func() {
		It("Should register the resolver", func() {
			Expect(Name()).To(Equal("Default"))
			RegisterTargetResolver(mockresolver)
			Expect(Name()).To(Equal("Mock Resolver"))
		})
	})

	Describe("Targets", func() {
		It("Should handle no resolver", func() {
			resolver = nil
			t, err := Targets(ctx, log)
			Expect(err).To(MatchError("no Provisioning Target Resolver registered"))
			Expect(t).To(Equal([]srvcache.Server{}))
		})

		It("Should handle empty response from the resolver", func() {
			build.ProvisionBrokerURLs = ""
			t, err := Targets(ctx, log)
			Expect(err).To(MatchError("provisioning target plugin Default returned no servers"))
			Expect(t).To(Equal([]srvcache.Server{}))
		})

		It("Should handle invalid format hosts", func() {
			build.ProvisionBrokerURLs = "foo,bar"
			t, err := Targets(ctx, log)
			Expect(err).To(MatchError("could not determine provisioning servers using Default provisionig target plugin: could not parse host foo: address foo: missing port in address"))
			Expect(t).To(Equal([]srvcache.Server{}))
		})

		It("Should handle valid format hosts", func() {
			build.ProvisionBrokerURLs = "foo:4222, nats://bar:4222"
			t, err := Targets(ctx, log)
			Expect(err).ToNot(HaveOccurred())
			Expect(t).To(Equal([]srvcache.Server{
				srvcache.Server{Host: "foo", Port: 4222, Scheme: "nats"},
				srvcache.Server{Host: "bar", Port: 4222, Scheme: "nats"},
			}))
		})
	})
})
