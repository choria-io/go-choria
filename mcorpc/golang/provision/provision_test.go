package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"testing"
)

func Test(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "McoRPC/Golang/Provision")
}

var _ = Describe("McoRPC/Golang/Provision", func() {
	var (
		mockctl   *gomock.Controller
		requests  chan *choria.ConnectorMessage
		cfg       *config.Config
		fw        *choria.Framework
		am        *agents.Manager
		err       error
		prov      *mcorpc.Agent
		reply     *mcorpc.Reply
		ctx       context.Context
		targetcfg string
		targetlog string
		targetdir string
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())

		requests = make(chan *choria.ConnectorMessage)
		reply = &mcorpc.Reply{}

		cfg, err = config.NewDefaultConfig()
		Expect(err).ToNot(HaveOccurred())
		cfg.DisableTLS = true

		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		am = agents.New(requests, fw, nil, agents.NewMockServerInfoSource(mockctl), logrus.WithFields(logrus.Fields{"test": "1"}))
		prov, err = New(am)
		Expect(err).ToNot(HaveOccurred())
		logrus.SetLevel(logrus.FatalLevel)

		allowRestart = false
		build.ProvisionModeDefault = "false"
		build.ProvisionBrokerURLs = "nats://n1:4222"
		ctx = context.Background()

		targetdir, err = ioutil.TempDir("", "provision_test")
		Expect(err).ToNot(HaveOccurred())

		targetcfg = filepath.Join(targetdir, "choria_test.cfg")
		targetlog = filepath.Join(targetdir, "choria_test.log")
	})

	AfterEach(func() {
		os.RemoveAll(targetdir)
		mockctl.Finish()
	})

	Describe("New", func() {
		It("Should create all the actions", func() {
			Expect(prov.ActionNames()).To(Equal([]string{"configure", "gencsr", "reprovision", "restart"}))
		})
	})

	Describe("csrAction", func() {
		It("Should only be active in provision mode", func() {
			csrAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot reconfigure a server that is not in provisioning mode"))
		})

		It("Should only be active when a configfile or ssl dir is set", func() {
			prov.Config.ConfigFile = ""
			prov.Config.Choria.SSLDir = ""
			build.ProvisionModeDefault = "true"

			csrAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot determine where to store SSL data, no configure file given and no SSL directory configured"))
		})

		It("Should create the Key, CSR and return the CSR", func() {
			prov.Config.Choria.SSLDir = filepath.Join(targetdir, "ssl")

			build.ProvisionModeDefault = "true"
			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"cn":"ginkgo.example.net"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			csrAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			csrr := reply.Data.(*CSRReply)
			Expect(csrr.SSLDir).To(Equal(filepath.Join(targetdir, "ssl")))
			stat, err := os.Stat(filepath.Join(prov.Config.Choria.SSLDir, "private.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(stat.Mode()).To(Equal(os.FileMode(0700)))
			stat, err = os.Stat(filepath.Join(prov.Config.Choria.SSLDir, "csr.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(stat.Mode()).To(Equal(os.FileMode(0700)))
		})
	})

	Describe("restartAction", func() {
		It("Should only restart nodes in provision mode", func() {
			restartAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot restart a server that is not in provisioning mode"))
		})

		It("Should refuse to restart nodes that just goes back into provision mode", func() {
			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = "testdata/provisioning.cfg"

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"splay":10}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			restartAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Configuration testdata/provisioning.cfg enables provisioning, restart cannot continue"))
		})

		It("Should restart with splay", func() {
			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = "testdata/default.cfg"

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"splay":10}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			restartAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(Reply).Message).To(MatchRegexp("Restarting Choria Server after \\d+s"))
		})
	})

	Describe("reprovisionAction", func() {
		It("Should only reprovision nodes not in provisioning mode", func() {
			build.ProvisionModeDefault = "true"

			reprovisionAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Server is already in provisioning mode, cannot enable provisioning mode again"))
		})

		It("Should fail when the config file cannot be determined", func() {
			cfg.ConfigFile = ""
			reprovisionAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot determine the configuration file to manage"))
		})

		It("Should write a sane config file without registration by default", func() {
			cfg.ConfigFile = targetcfg

			reprovisionAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))

			cfg, err := config.NewConfig(targetcfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Choria.Provision).To(BeTrue())
			Expect(cfg.Choria.FileContentRegistrationData).To(BeEmpty())
			Expect(cfg.Choria.FileContentRegistrationTarget).To(BeEmpty())
			Expect(cfg.LogFile).To(BeEmpty())
		})

		It("Should support setting a logfile and file_content registration", func() {
			cfg.ConfigFile = targetcfg
			cfg.LogFile = targetlog
			cfg.LogLevel = "info"
			cfg.Registration = []string{"file_content"}
			cfg.Choria.FileContentRegistrationData = "/tmp/choria_test.json"
			cfg.Choria.FileContentRegistrationTarget = "default.registration"

			reprovisionAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))

			cfg, err := config.NewConfig(targetcfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Choria.Provision).To(BeTrue())
			Expect(cfg.LogLevel).To(Equal("debug"))
			Expect(cfg.LogFile).To(Equal(targetlog))
			Expect(cfg.Registration).To(Equal([]string{"file_content"}))
			Expect(cfg.Choria.FileContentRegistrationData).To(Equal("/tmp/choria_test.json"))
		})
	})

	Describe("configureAction", func() {
		It("Should only allow configuration when in provision mode", func() {
			cfg.Choria.Provision = false

			configureAction(ctx, &mcorpc.Request{}, reply, prov, nil)

			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot reconfigure a server that is not in provisioning mode"))
		})

		It("Should fail for unknown config files", func() {
			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = ""

			configureAction(ctx, &mcorpc.Request{}, reply, prov, nil)

			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot determine the configuration file to manage"))
		})

		It("Should fail for empty configuration", func() {
			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = targetcfg

			configureAction(ctx, &mcorpc.Request{Data: json.RawMessage("{}")}, reply, prov, nil)

			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Did not receive any configuration to write, cannot write a empty configuration file"))
		})

		It("Should write the configuration", func() {
			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = targetcfg

			req := &mcorpc.Request{
				Data:      json.RawMessage(fmt.Sprintf(`{"certificate": "stub_cert", "ca":"stub_ca", "ssldir":"%s", "config":"{\"plugin.choria.server.provision\":\"0\", \"plugin.choria.srv_domain\":\"another.com\"}"}`, targetdir)),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			Expect(targetcfg).ToNot(BeAnExistingFile())
			configureAction(ctx, req, reply, prov, nil)

			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(Reply).Message).To(Equal(fmt.Sprintf("Wrote 3 lines to %s", targetcfg)))
			Expect(targetcfg).To(BeAnExistingFile())

			cfg, err := config.NewConfig(targetcfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Choria.SRVDomain).To(Equal("another.com"))

			cert, err := ioutil.ReadFile(filepath.Join(targetdir, "certificate.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(cert)).To(Equal("stub_cert"))

			ca, err := ioutil.ReadFile(filepath.Join(targetdir, "ca.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(ca)).To(Equal("stub_ca"))
		})
	})
})
