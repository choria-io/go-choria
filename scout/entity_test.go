package scout

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registration")
}

var _ = Describe("Entity", func() {
	var (
		log     *logrus.Entry
		mockctl *gomock.Controller
		fw      *MockFramework
	)

	BeforeEach(func() {
		logger := logrus.New()
		logger.SetOutput(GinkgoWriter)
		log = logrus.NewEntry(logger)
		logrus.SetOutput(GinkgoWriter)
		mockctl = gomock.NewController(GinkgoT())
		fw = NewMockFramework(mockctl)

		fw.EXPECT().Identity().Return("ginkgo.example.net").AnyTimes()
		fw.EXPECT().Logger(gomock.Any()).Return(log).AnyTimes()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	It("Should get all checks", func() {
		srv, err := server.NewServer(&server.Options{JetStream: true, Port: -1})
		Expect(err).ToNot(HaveOccurred())
		go srv.Start()
		if !srv.ReadyForConnections(5 * time.Second) {
			Fail("jetstream did not become ready")
		}

		nc, err := nats.Connect(srv.ClientURL())
		Expect(err).ToNot(HaveOccurred())

		err = ConfigureStreams(nc, log)
		Expect(err).ToNot(HaveOccurred())

		_, err = nc.Request("scout.tags.common", []byte(`["check_puppet", "check_backups"]`), time.Second)
		Expect(err).ToNot(HaveOccurred())

		_, err = nc.Request("scout.tags.ginkgo.example.net", []byte(`["check_specific1", "check_specific2", "swap"]`), time.Second)
		Expect(err).ToNot(HaveOccurred())

		_, err = nc.Request("scout.overrides.ginkgo.example.net", []byte(`{"check_load":{"crit":10,"warn":5}}`), time.Second)
		Expect(err).ToNot(HaveOccurred())

		load, err := ioutil.ReadFile("testdata/swap.json")
		Expect(err).ToNot(HaveOccurred())

		_, err = nc.Request("scout.check.swap", load, 2*time.Second)
		Expect(err).ToNot(HaveOccurred())

		names, err := jsm.StreamNames(jsm.WithConnection(nc))
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("%v\n", names)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		wg := &sync.WaitGroup{}

		otf, err := ioutil.TempFile("", "")
		Expect(err).ToNot(HaveOccurred())
		otf.Close()
		defer os.Remove(otf.Name())

		mtf, err := ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		fw.EXPECT().NATSConn().Return(nc).AnyTimes()
		fw.EXPECT().ScoutOverridesFile().Return(otf.Name()).AnyTimes()
		fw.EXPECT().ScoutTags().Return([]string{"common", "ginkgo.example.net"}).AnyTimes()
		fw.EXPECT().MachineSourceDir().Return(mtf).AnyTimes()

		entity, err := NewEntity(ctx, wg, fw)
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(time.Second)

		Expect(entity.checkNames()).To(Equal([]string{"check_backups", "check_puppet", "check_specific1", "check_specific2", "swap"}))

		oj, err := ioutil.ReadFile(otf.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(oj).To(MatchJSON(`{"check_load":{"crit":10, "warn": 5}}`))

		_, err = nc.Request("scout.overrides.ginkgo.example.net", []byte(`{"check_load":{"crit":20,"warn":10}}`), time.Second)
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(time.Second)

		oj, err = ioutil.ReadFile(otf.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(oj).To(MatchJSON(`{"check_load":{"crit":20, "warn": 10}}`))

		log.Infof("Removing swap tag")
		_, err = nc.Request("scout.tags.ginkgo.example.net", []byte(`["check_specific1", "check_specific2"]`), time.Second)
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(2 * time.Second)
	})
})
