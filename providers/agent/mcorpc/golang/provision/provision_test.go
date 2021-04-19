package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/lifecycle"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-updater"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func Test(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provision/Agent")
}

var _ = Describe("Provision/Agent", func() {
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
		si        *MockServerInfoSource
		targetcfg string
		targetlog string
		targetdir string
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())

		requests = make(chan *choria.ConnectorMessage)
		reply = &mcorpc.Reply{}

		cfg = config.NewConfigForTests()
		Expect(err).ToNot(HaveOccurred())
		cfg.DisableTLS = true
		cfg.InitiatedByServer = true
		cfg.LogLevel = "warn"

		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		si = NewMockServerInfoSource(mockctl)
		am = agents.New(requests, fw, nil, si, logrus.WithFields(logrus.Fields{"test": "1"}))
		p, err := New(am)
		Expect(err).ToNot(HaveOccurred())
		prov = p.(*mcorpc.Agent)
		prov.SetServerInfo(si)
		logrus.SetLevel(logrus.FatalLevel)

		allowRestart = false
		SetRestartAction(restart)
		build.ProvisionModeDefault = "false"
		build.ProvisionBrokerURLs = "nats://n1:4222"
		build.ProvisionToken = ""

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
			Expect(prov.ActionNames()).To(Equal([]string{"configure", "gencsr", "jwt", "release_update", "reprovision", "restart"}))
		})
	})

	Describe("releaseUpdateAction", func() {
		It("should require a token", func() {
			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"token":"toomanysecrets"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			build.ProvisionToken = "xx"
			releaseUpdateAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
		})

		It("Should handle update errors", func() {
			updaterf = func(_ ...updater.Option) error {
				return errors.New("simulated eror")
			}

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"token":"toomanysecrets", "version":"0.7.0"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			build.ProvisionToken = "toomanysecrets"
			releaseUpdateAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Update to version 0.7.0 failed, release rolled back: simulated eror"))
		})

		It("Should update and publish an event", func() {
			updaterf = func(_ ...updater.Option) error {
				return nil
			}

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"token":"toomanysecrets", "version":"0.7.0"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			build.ProvisionToken = "toomanysecrets"

			si.EXPECT().NewEvent(lifecycle.Shutdown).Times(1)
			releaseUpdateAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Statusmsg).To(Equal(""))
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

		It("Should support a token", func() {
			prov.Config.Choria.SSLDir = filepath.Join(targetdir, "ssl")
			build.ProvisionToken = "fail"
			build.ProvisionModeDefault = "true"

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"cn":"ginkgo.example.net", "token":"toomanysecrets"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			csrAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))

			build.ProvisionToken = "toomanysecrets"
			reply = &mcorpc.Reply{}

			csrAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
		})

		It("Should create the Key, CSR and return the CSR", func() {
			// TODO: windows support
			if runtime.GOOS == "windows" {
				Skip("TODO: windows support")
			}

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

			// TODO: windows
			if runtime.GOOS != "windows" {
				Expect(stat.Mode()).To(Equal(os.FileMode(0640)))
			}

			stat, err = os.Stat(filepath.Join(prov.Config.Choria.SSLDir, "csr.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(stat.Mode()).To(Equal(os.FileMode(0644)))
		})
	})

	Describe("restartAction", func() {
		It("Should not restart nodes not provision mode", func() {
			build.ProvisionToken = ""
			restartAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot restart a server that is not in provisioning mode or with no token set"))
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
			SetRestartAction(restart)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Configuration testdata/provisioning.cfg enables provisioning, restart cannot continue"))
		})

		It("Should support a token", func() {
			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = "testdata/default.cfg"
			build.ProvisionToken = "fail"

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"splay":10, "token":"toomanysecrets"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			restartAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))

			// tests the path with no provisioning set but with a token set
			build.ProvisionModeDefault = "false"
			build.ProvisionToken = "toomanysecrets"
			reply = &mcorpc.Reply{}

			Expect(prov.Choria.ProvisionMode()).To(BeFalse())
			restartAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))

			// tests the path with provision mode and no token
			build.ProvisionModeDefault = "true"
			build.ProvisionToken = ""
			reply = &mcorpc.Reply{}

			Expect(prov.Choria.ProvisionMode()).To(BeTrue())
			restartAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
		})

		It("Should restart with splay", func() {
			// TODO: windows support
			if runtime.GOOS == "windows" {
				Skip("TODO: windows support")
			}

			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = "testdata/default.cfg"

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"splay":10}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			didRestart := false
			SetRestartAction(func(_ time.Duration, _ agents.ServerInfoSource, _ *logrus.Entry) {
				didRestart = true
			})

			restartAction(ctx, req, reply, prov, nil)
			runtime.Gosched()

			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(Reply).Message).To(MatchRegexp("Restarting Choria Server after \\d+s"))
			Expect(didRestart).To(BeTrue())
		})
	})

	Describe("reprovisionAction", func() {
		var req *mcorpc.Request

		BeforeEach(func() {
			req = &mcorpc.Request{
				Data: json.RawMessage(`{}`),
			}
		})

		It("Should only reprovision nodes not in provisioning mode", func() {
			build.ProvisionModeDefault = "true"

			reprovisionAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Server is already in provisioning mode, cannot enable provisioning mode again"))
		})

		It("Should fail when the config file cannot be determined", func() {
			cfg.ConfigFile = ""
			reprovisionAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot determine the configuration file to manage"))
		})

		It("Should fail for wrong tokens with not an empty token", func() {
			cfg.ConfigFile = targetcfg
			build.ProvisionToken = "toomanysecrets"

			req.Data = json.RawMessage(`{"token":"fail"}`)

			reprovisionAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Incorrect provision token supplied"))
		})

		It("Should match tokens", func() {
			cfg.ConfigFile = targetcfg
			build.ProvisionToken = "toomanysecrets"

			req.Data = json.RawMessage(`{"token":"toomanysecrets"}`)

			reprovisionAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
		})

		It("Should write a sane config file without registration by default", func() {
			cfg.ConfigFile = targetcfg
			build.ProvisionToken = ""

			reprovisionAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))

			cfg, err := config.NewConfig(targetcfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Choria.Provision).To(BeTrue())
			Expect(cfg.Choria.FileContentRegistrationData).To(BeEmpty())
			Expect(cfg.Choria.FileContentRegistrationTarget).To(BeEmpty())
			Expect(cfg.LogFile).To(Equal("discard"))
		})

		It("Should support setting a logfile and file_content registration", func() {
			cfg.ConfigFile = targetcfg
			cfg.LogFile = targetlog
			cfg.LogLevel = "info"
			cfg.Registration = []string{"file_content"}
			cfg.Choria.FileContentRegistrationData = "/tmp/choria_test.json"
			cfg.Choria.FileContentRegistrationTarget = "default.registration"
			build.ProvisionRegistrationData = ""

			reprovisionAction(ctx, req, reply, prov, nil)
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

	Describe("jwtAction", func() {
		It("Should require a token", func() {
			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"token":"toomanysecrets"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			build.ProvisionToken = "fail"
			build.ProvisionJWTFile = "testdata/provision.jwt"

			jwtAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Incorrect provision token supplied"))

			build.ProvisionToken = "toomanysecrets"
			reply = &mcorpc.Reply{}

			jwtAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Statusmsg).To(Equal(""))
		})

		It("Should handle unset JWT files", func() {
			build.ProvisionToken = ""
			build.ProvisionJWTFile = ""

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			jwtAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("No Provisioning JWT file has been configured"))
		})

		It("Should handle missing JWT files", func() {
			build.ProvisionToken = ""
			build.ProvisionJWTFile = "/nonexisting"

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			jwtAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Provisioning JWT file does not exist"))
		})

		It("Should read the JWT file", func() {
			build.ProvisionToken = ""
			build.ProvisionJWTFile = "testdata/provision.jwt"

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			jwtAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(JWTReply).JWT).To(Equal("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjaHMiOnRydWUsImNodSI6InByb3YuZXhhbXBsZS5uZXQ6NDIyMiIsImNodCI6InNlY3JldCIsImNocGQiOnRydWV9.lLc9DAdjkdA-YAbhwHg3FVR9BklGFSZ7FxyzSbh9vCc"))
			build.ProvisionJWTFile = ""
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

		It("Should support a token", func() {
			// TODO: windows support
			if runtime.GOOS == "windows" {
				Skip("TODO: windows support")
			}

			build.ProvisionToken = "fail"
			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = targetcfg

			req := &mcorpc.Request{
				Data:      json.RawMessage(fmt.Sprintf(`{"token":"toomanysecrets", "certificate": "stub_cert", "ca":"stub_ca", "ssldir":"%s", "config":"{\"plugin.choria.server.provision\":\"0\", \"plugin.choria.srv_domain\":\"another.com\"}"}`, targetdir)),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			configureAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))

			build.ProvisionToken = "toomanysecrets"
			reply = &mcorpc.Reply{}

			si.EXPECT().NewEvent(lifecycle.Provisioned).Times(1)

			configureAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
		})

		It("Should write the configuration", func() {
			// TODO: windows support
			if runtime.GOOS == "windows" {
				Skip("TODO: windows support")
			}

			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = targetcfg

			req := &mcorpc.Request{
				Data:      json.RawMessage(fmt.Sprintf(`{"certificate": "stub_cert", "ca":"stub_ca", "ssldir":"%s", "config":"{\"plugin.choria.server.provision\":\"0\", \"plugin.choria.srv_domain\":\"another.com\"}"}`, targetdir)),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			si.EXPECT().NewEvent(lifecycle.Provisioned).Times(1)

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
