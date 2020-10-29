package scout

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registration")
}

var _ = Describe("Entity", func() {
	var (
		fw  *choria.Framework
		log *logrus.Entry
		err error
	)

	BeforeEach(func() {
		cfg := config.NewConfigForTests()
		cfg.DisableSecurityProviderVerify = true
		cfg.DisableTLS = true
		cfg.Choria.ScoutTags = "testdata/tags.json"
		cfg.OverrideCertname = "ginkgo.example.net"

		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		fw.SetLogWriter(GinkgoWriter)
		log = fw.Logger("ginkgo")
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

		mgr, err := jsm.New(nc)
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

		names, err := mgr.StreamNames()
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("%v\n", names)

		otf, err := ioutil.TempFile("", "")
		Expect(err).ToNot(HaveOccurred())
		otf.Close()
		defer os.Remove(otf.Name())

		mtf, err := ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		fw.Config.Choria.MachineSourceDir = mtf
		fw.Config.Choria.ScoutOverrides = otf.Name()

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		wg := &sync.WaitGroup{}

		fw.Config.Choria.MiddlewareHosts = []string{srv.ClientURL()}

		scout, err := New(fw)
		Expect(err).ToNot(HaveOccurred())
		err = scout.Start(ctx, wg, false)
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(time.Second)

		Expect(scout.entity.checkNames()).To(Equal([]string{"check_backups", "check_puppet", "check_specific1", "check_specific2", "swap"}))

		oj, err := ioutil.ReadFile(otf.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(oj).To(MatchJSON(`{"check_load":{"crit":10, "warn": 5}}`))

		_, err = nc.Request("scout.overrides.ginkgo.example.net", []byte(`{"check_load":{"crit":20,"warn":10}}`), time.Second)
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(time.Second)

		oj, err = ioutil.ReadFile(otf.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(oj).To(MatchJSON(`{"check_load":{"crit":20, "warn": 10}}`))

		_, err = nc.Request("scout.tags.ginkgo.example.net", []byte(`["check_specific1", "check_specific2"]`), time.Second)
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(2 * time.Second)
	})
})
