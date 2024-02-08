// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/lifecycle"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-updater"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provision/Agent")
}

var _ = Describe("Provision/Agent", func() {
	var (
		mockctl   *gomock.Controller
		requests  chan inter.ConnectorMessage
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

		requests = make(chan inter.ConnectorMessage)
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

		targetdir, err = os.MkdirTemp("", "provision_test")
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
			Expect(prov.ActionNames()).To(Equal([]string{"configure", "gen25519", "gencsr", "jwt", "release_update", "reprovision", "restart", "shutdown"}))
		})
	})

	Describe("lockedAction", func() {
		It("Should obtain a lock and not let others interact while locked", func() {
			build.ProvisionToken = ""
			LockWindow = 500 * time.Millisecond

			// first call will lock it for 2 minutes, we reach the wrapped action that produce a error
			lockedAction(shutdownAction)(ctx, &mcorpc.Request{CallerID: "one", SenderID: "ginkgo"}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot shutdown a server that is not in provisioning mode or with no token set"))
			Expect(lockedUntil).To(BeTemporally("~", time.Now().Add(LockWindow), time.Millisecond))
			Expect(lockedBy).To(Equal("ginkgo.one"))

			// now someone else calls and we expect locked error
			lockedAction(shutdownAction)(ctx, &mcorpc.Request{CallerID: "two", SenderID: "ginkgo"}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal(LockedError))

			// but when we call we expect it to work and get through to the wrapped action
			lockedAction(shutdownAction)(ctx, &mcorpc.Request{CallerID: "one", SenderID: "ginkgo"}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot shutdown a server that is not in provisioning mode or with no token set"))

			// now we let the lock window pass, we expect another caller to be able to reach the server
			time.Sleep(500 * time.Millisecond)
			lockedAction(shutdownAction)(ctx, &mcorpc.Request{CallerID: "two", SenderID: "ginkgo"}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot shutdown a server that is not in provisioning mode or with no token set"))
		})
	})

	Describe("ed25519Action", func() {
		var td string

		BeforeEach(func() {
			td, err = os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(td)
		})

		It("should require a token", func() {
			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"token":"toomanysecrets"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			build.ProvisionToken = "xx"
			ed25519Action(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
		})

		It("Should create the token in the right location and sign the nonce", func() {
			j, err := json.Marshal(&ED25519Request{Nonce: "nonce", Token: "toomanysecrets"})
			Expect(err).ToNot(HaveOccurred())

			req := &mcorpc.Request{
				Data:      j,
				RequestID: "uid",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			build.ProvisionModeDefault = "true"
			build.ProvisionToken = "toomanysecrets"

			cfg.ConfigFile = filepath.Join(td, "server.conf")
			jwtPath := filepath.Join(td, "server.jwt")
			Expect(os.WriteFile(jwtPath, []byte("x"), 0700)).To(Succeed())

			ed25519Action(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(iu.FileExist(jwtPath)).To(BeFalse())

			resp := reply.Data.(*ED25519Reply)
			Expect(resp.PublicKey).To(HaveLen(64))
			pubKBytes, err := hex.DecodeString(resp.PublicKey)
			Expect(err).ToNot(HaveOccurred())
			sig, err := hex.DecodeString(resp.Signature)
			Expect(err).ToNot(HaveOccurred())
			Expect(ed25519.Verify(pubKBytes, []byte("nonce"), sig)).To(BeTrue())
			Expect(resp.Directory).To(Equal(td))

			_, priK, err := choria.Ed25519KeyPairFromSeedFile(filepath.Join(td, "server.seed"))
			Expect(err).ToNot(HaveOccurred())
			Expect(priK.Public()).To(Equal(ed25519.PublicKey(pubKBytes)))
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
			Expect(stat.Mode()).To(Equal(os.FileMode(0600)))
			stat, err = os.Stat(filepath.Join(prov.Config.Choria.SSLDir, "csr.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(stat.Mode()).To(Equal(os.FileMode(0644)))
		})
	})

	Describe("shutdownAction", func() {
		It("Should not shut nodes not provision mode", func() {
			build.ProvisionToken = ""
			shutdownAction(ctx, &mcorpc.Request{}, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Cannot shutdown a server that is not in provisioning mode or with no token set"))
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

			shutdownAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))

			// tests the path with no provisioning set but with a token set
			build.ProvisionModeDefault = "false"
			build.ProvisionToken = "toomanysecrets"
			reply = &mcorpc.Reply{}

			Expect(prov.Choria.ProvisionMode()).To(BeFalse())
			shutdownAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))

			// tests the path with provision mode and no token
			build.ProvisionModeDefault = "true"
			build.ProvisionToken = ""
			reply = &mcorpc.Reply{}

			Expect(prov.Choria.ProvisionMode()).To(BeTrue())
			restartAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
		})

		It("Should shutdown with splay", func() {
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

			done := make(chan bool)
			SetShutdownAction(func(_ time.Duration, _ agents.ServerInfoSource, _ *logrus.Entry) {
				close(done)
			})

			shutdownAction(ctx, req, reply, prov, nil)
			runtime.Gosched()
			<-done

			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(Reply).Message).To(MatchRegexp("Shutting Choria Server down after \\d+s"))
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

			done := make(chan bool)
			SetRestartAction(func(_ time.Duration, _ agents.ServerInfoSource, _ *logrus.Entry) {
				close(done)
			})

			restartAction(ctx, req, reply, prov, nil)
			runtime.Gosched()
			<-done

			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(Reply).Message).To(MatchRegexp("Restarting Choria Server after \\d+s"))
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

			build.ProvisionToken = "testdata/provisioning.cfg"
			build.ProvisionModeDefault = "true"
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
			build.ProvisionToken = "testdata/provisioning.cfg"
			build.ProvisionJWTFile = ""
			build.ProvisionModeDefault = "true"
			build.ProvisionToken = ""

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
			build.ProvisionToken = "testdata/provisioning.cfg"
			build.ProvisionJWTFile = "/nonexisting"
			build.ProvisionModeDefault = "true"
			build.ProvisionToken = ""

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
			cfg.ConfigFile = "testdata/provisioning.cfg"
			build.ProvisionModeDefault = "true"
			build.ProvisionToken = ""
			build.ProvisionJWTFile = "testdata/provision.jwt"

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			jwtAction(ctx, req, reply, prov, nil)
			Expect(reply.Statusmsg).To(Equal(""))
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

		It("Should require a EDCH key when a private key is provided", func() {
			if runtime.GOOS == "windows" {
				Skip("TODO: windows support")
			}

			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = targetcfg

			req := &mcorpc.Request{
				Data:      json.RawMessage(fmt.Sprintf(`{"certificate": "stub_cert", "ca":"stub_ca", "key":"stub_key","ssldir":"%s", "config":"{\"plugin.choria.server.provision\":\"0\", \"plugin.choria.srv_domain\":\"another.com\"}"}`, targetdir)),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			configureAction(ctx, req, reply, prov, nil)

			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("EDCH Public Key not supplied while providing a private key"))
		})

		It("Should write the configuration", func() {
			// TODO: windows support
			if runtime.GOOS == "windows" {
				Skip("TODO: windows support")
			}

			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = targetcfg

			roguePolicy := filepath.Join(targetdir, "policies", "rogue.policy")
			rogueRego := filepath.Join(targetdir, "policies", "rego", "rogue.rego")

			err = os.MkdirAll(filepath.Join(targetdir, "policies"), 0700)
			Expect(err).ToNot(HaveOccurred())
			err = os.MkdirAll(filepath.Join(targetdir, "policies", "rego"), 0700)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(roguePolicy, []byte("policy default allow"), 0400)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(rogueRego, []byte("package io.choria.mcorpc.authpolicy"), 0400)

			// provisioner pub: c25fab933b16b2a4589e2c2d297669b818c82881f2f9bfd08cb8fd298924a41b
			// provisioner pri: 3e150feab1f2c709af9afa28e626f258dbf1b683d3bc35225a9b0489de485870
			// server pub: 88a9a0ed27dc93c29466ea2bef99e078342b27e7a1d789fc35a9131f86c3a022
			// server pri: 67e4a9b3934a3030470ed7a30f89eeaf7dab7b492aa9ee02fb864d690b7e6eeb
			// shared: 53a4887b448755a3f0a0618b289eb2369ec4b2461d44890abdfa3084ef080c55
			ecdhPrivate, err = hex.DecodeString("67e4a9b3934a3030470ed7a30f89eeaf7dab7b492aa9ee02fb864d690b7e6eeb")
			Expect(err).ToNot(HaveOccurred())

			ecdhPublic, err = hex.DecodeString("88a9a0ed27dc93c29466ea2bef99e078342b27e7a1d789fc35a9131f86c3a022")
			Expect(err).ToNot(HaveOccurred())

			shared, err := ecdhSharedSecretLocked("c25fab933b16b2a4589e2c2d297669b818c82881f2f9bfd08cb8fd298924a41b")
			Expect(err).ToNot(HaveOccurred())

			Expect(fmt.Sprintf("%x", shared)).To(Equal("53a4887b448755a3f0a0618b289eb2369ec4b2461d44890abdfa3084ef080c55"))

			pk, err := rsa.GenerateKey(rand.Reader, 1024)
			Expect(err).ToNot(HaveOccurred())
			pkBytes := x509.MarshalPKCS1PrivateKey(pk)
			pkPem := &bytes.Buffer{}
			err = pem.Encode(pkPem, &pem.Block{Bytes: pkBytes, Type: "RSA PRIVATE KEY"})
			Expect(err).ToNot(HaveOccurred())

			epb, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(pk), shared, x509.PEMCipherAES256) //lint:ignore SA1019 there is no alternative
			Expect(err).ToNot(HaveOccurred())
			epbPem := &bytes.Buffer{}
			err = pem.Encode(epbPem, epb)
			Expect(err).ToNot(HaveOccurred())

			data := ConfigureRequest{
				Certificate:   "stub_cert",
				CA:            "stub_ca",
				SSLDir:        targetdir,
				Configuration: "{\"plugin.choria.server.provision\":\"0\", \"plugin.choria.srv_domain\":\"another.com\"}",
				ECDHPublic:    "c25fab933b16b2a4589e2c2d297669b818c82881f2f9bfd08cb8fd298924a41b", // provisioner sends this, so this is its public key
				Key:           epbPem.String(),                                                    // encrypted using shared of the EDCH
				ActionPolicies: map[string]string{
					"choria_util.policy": "policy default deny",
					"rpcutil.policy":     "policy default allow",
				},
				OPAPolicies: map[string]string{
					"default.rego": "package io.choria.mcorpc.authpolicy",
				},
			}

			jdat, _ := json.Marshal(data)
			req := &mcorpc.Request{
				Data:      jdat,
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			si.EXPECT().NewEvent(lifecycle.Provisioned).Times(1)

			Expect(targetcfg).ToNot(BeAnExistingFile())
			configureAction(ctx, req, reply, prov, nil)

			Expect(reply.Statusmsg).To(Equal(""))
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(Reply).Message).To(Equal(fmt.Sprintf("Wrote 3 lines to %s", targetcfg)))
			Expect(targetcfg).To(BeAnExistingFile())

			cfg, err := config.NewConfig(targetcfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Choria.SRVDomain).To(Equal("another.com"))

			cert, err := os.ReadFile(filepath.Join(targetdir, "certificate.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(cert)).To(Equal("stub_cert"))

			ca, err := os.ReadFile(filepath.Join(targetdir, "ca.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(ca)).To(Equal("stub_ca"))

			key, err := os.ReadFile(filepath.Join(targetdir, "private.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(key).To(Equal(pkPem.Bytes()))

			Expect(filepath.Join(targetdir, "csr.pem")).ToNot(BeAnExistingFile())

			policy, err := os.ReadFile(filepath.Join(targetdir, "policies", "choria_util.policy"))
			Expect(err).ToNot(HaveOccurred())
			Expect(policy).To(Equal([]byte("policy default deny")))

			policy, err = os.ReadFile(filepath.Join(targetdir, "policies", "rpcutil.policy"))
			Expect(err).ToNot(HaveOccurred())
			Expect(policy).To(Equal([]byte("policy default allow")))

			policy, err = os.ReadFile(filepath.Join(targetdir, "policies", "rego", "default.rego"))
			Expect(err).ToNot(HaveOccurred())
			Expect(policy).To(Equal([]byte("package io.choria.mcorpc.authpolicy")))

			Expect(roguePolicy).ToNot(BeAnExistingFile())
			Expect(rogueRego).ToNot(BeAnExistingFile())
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

		It("Should require being enabled in the configuration file", func() {
			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"token":"toomanysecrets"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}
			build.ProvisionToken = "xx"
			cfg.ConfigFile = "testdata/provisioning.cfg"
			build.ProvisionModeDefault = "true"

			releaseUpdateAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(reply.Statusmsg).To(Equal("Version updates is not enabled in the configuration"))

			build.ProvisionModeDefault = "false"
		})

		It("Should handle update errors", func() {
			build.ProvisionModeDefault = "true"
			cfg.ConfigFile = "testdata/provisioning.cfg"
			cfg.Choria.ProvisionAllowUpdate = true

			updaterf = func(_ ...updater.Option) error {
				return errors.New("simulated error")
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
			Expect(reply.Statusmsg).To(Equal("Update to version 0.7.0 failed, release rolled back: simulated error"))
		})

		It("Should update and publish an event", func() {
			build.ProvisionToken = "testdata/provisioning.cfg"
			build.ProvisionModeDefault = "true"
			build.ProvisionToken = "toomanysecrets"
			cfg.Choria.ProvisionAllowUpdate = true

			updaterf = func(_ ...updater.Option) error {
				return nil
			}

			req := &mcorpc.Request{
				Data:      json.RawMessage(`{"token":"toomanysecrets", "version":"0.7.0"}`),
				RequestID: "uniq_req_id",
				CallerID:  "choria=rip.mcollective",
				SenderID:  "go.test",
			}

			si.EXPECT().NewEvent(gomock.Any(), gomock.Any()).Do(func(e lifecycle.Type, opts ...lifecycle.Option) error {
				event, err := lifecycle.New(e, opts...)
				Expect(err).ToNot(HaveOccurred())
				Expect(event.Type()).To(Equal(lifecycle.Upgraded))
				Expect(event.(*lifecycle.UpgradedEvent).NewVersion).To(Equal("0.7.0"))
				return nil
			})
			releaseUpdateAction(ctx, req, reply, prov, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Statusmsg).To(Equal(""))
		})
	})
})
