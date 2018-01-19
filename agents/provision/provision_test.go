package provision

import (
	"context"
	"encoding/json"
	"os"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/mcorpc"
	"github.com/choria-io/go-choria/server/agents"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"testing"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent/Provision")
}

var _ = Describe("Agent/Provision", func() {
	var (
		requests chan *choria.ConnectorMessage
		cfg      *choria.Config
		fw       *choria.Framework
		am       *agents.Manager
		err      error
		prov     *mcorpc.Agent
		reply    *mcorpc.Reply
		ctx      context.Context
	)

	BeforeEach(func() {
		requests = make(chan *choria.ConnectorMessage)
		reply = &mcorpc.Reply{}

		cfg, err = choria.NewConfig("/dev/null")
		Expect(err).ToNot(HaveOccurred())
		cfg.DisableTLS = true

		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		am = agents.New(requests, fw, nil, logrus.WithFields(logrus.Fields{"test": "1"}))
		prov, err = New(am)
		Expect(err).ToNot(HaveOccurred())
		logrus.SetLevel(logrus.FatalLevel)

		allowRestart = false
		build.ProvisionModeDefault = "false"
		build.ProvisionBrokerURLs = "nats://n1:4222"
		ctx = context.Background()
	})

	AfterEach(func() {
		os.Remove("/tmp/choria_test.cfg")
	})

	var _ = Describe("New", func() {
		It("Should create all the actions", func() {
			Expect(prov.ActionNames()).To(Equal([]string{"configure", "reprovision", "restart"}))
		})
	})

	var _ = Describe("restartAction", func() {
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

	var _ = Describe("reprovisionAction", func() {
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
			cfg.ConfigFile = "/tmp/choria_test.cfg"

			reprovisionAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))

			cfg, err := choria.NewConfig("/tmp/choria_test.cfg")
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Choria.Provision).To(BeTrue())
			Expect(cfg.Choria.FileContentRegistrationData).To(BeEmpty())
			Expect(cfg.Choria.FileContentRegistrationTarget).To(BeEmpty())
			Expect(cfg.LogFile).To(BeEmpty())
		})

		It("Should support setting a logfile and file_content registration", func() {
			cfg.ConfigFile = "/tmp/choria_test.cfg"
			cfg.LogFile = "/tmp/choria_test.log"
			cfg.LogLevel = "info"
			cfg.Registration = []string{"file_content"}
			cfg.Choria.FileContentRegistrationData = "/tmp/choria_test.json"
			cfg.Choria.FileContentRegistrationTarget = "default.registration"

			reprovisionAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))

			cfg, err := choria.NewConfig("/tmp/choria_test.cfg")
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Choria.Provision).To(BeTrue())
			Expect(cfg.LogLevel).To(Equal("debug"))
			Expect(cfg.LogFile).To(Equal("/tmp/choria_test.log"))
			Expect(cfg.Registration).To(Equal([]string{"file_content"}))
			Expect(cfg.Choria.FileContentRegistrationData).To(Equal("/tmp/choria_test.json"))
		})
	})

	var _ = Describe("configureAction", func() {
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
			cfg.ConfigFile = "/tmp/choria_test.cfg"

			configureAction(ctx, &mcorpc.Request{Data: json.RawMessage("{}")}, reply, prov, nil)

			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Did not receive any configuration to write, cannot write a empty configuration file"))
		})

		It("Should write the configuration", func() {
			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = "/tmp/choria_test.cfg"

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"config":{"plugin.choria.server.provision":"0", "plugin.choria.srv_domain":"another.com"}}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			Expect("/tmp/choria_test.cfg").ToNot(BeAnExistingFile())
			configureAction(ctx, req, reply, prov, nil)

			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(Reply).Message).To(Equal("Wrote 3 lines to /tmp/choria_test.cfg"))
			Expect("/tmp/choria_test.cfg").To(BeAnExistingFile())

			cfg, err := choria.NewConfig("/tmp/choria_test.cfg")
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Choria.SRVDomain).To(Equal("another.com"))
		})
	})
})
