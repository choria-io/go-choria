// Copyright (c) 2020-2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/choria-io/go-choria/config"
	v1 "github.com/choria-io/go-choria/protocol/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Agent/McoRPC/Audit")
}

var _ = Describe("McoRPC/Audit", func() {
	var cfg *config.Config
	var err error

	BeforeEach(func() {
		if runtime.GOOS == "windows" {
			cfg, err = config.NewConfig("testdata/audit_windows.cfg")
		} else {
			cfg, err = config.NewConfig("testdata/audit.cfg")
		}
		Expect(err).ToNot(HaveOccurred())

		// use a temp file to ensure left over files dont cause problems
		tf, err := os.CreateTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(tf.Close()).To(Succeed())
		Expect(os.Remove(tf.Name())).To(Succeed())

		cfg.Choria.RPCAuditLogfile = tf.Name()

		DeferCleanup(func() {
			os.Remove(tf.Name())
		})
	})

	It("Should correctly audit the request", func() {
		Expect(cfg.RPCAudit).To(BeTrue())
		Expect(cfg.Choria.RPCAuditLogfile).ToNot(BeAnExistingFile())

		req, err := v1.NewRequest("test_agent", "test.node", "choria=rip.mcollective", 120, "uniq_req_id", "mcollective")
		Expect(err).ToNot(HaveOccurred())

		ok := Request(req, "test_agent", "test_action", json.RawMessage(`{"hello":"world"}`), cfg)
		Expect(ok).To(BeTrue())
		Expect(cfg.Choria.RPCAuditLogfile).To(BeAnExistingFile())

		j, err := os.ReadFile(cfg.Choria.RPCAuditLogfile)
		Expect(err).ToNot(HaveOccurred())

		am := Message{}
		err = json.Unmarshal(j, &am)
		Expect(err).ToNot(HaveOccurred())

		Expect(am.RequestID).To(Equal(req.RequestID()))
		Expect(am.RequestTime).To(Equal(req.Time().UTC().Unix()))
		Expect(am.CallerID).To(Equal("choria=rip.mcollective"))
		Expect(am.Sender).To(Equal("test.node"))
		Expect(am.Agent).To(Equal("test_agent"))
		Expect(am.Action).To(Equal("test_action"))
		Expect(am.Data).To(Equal(json.RawMessage(`{"hello":"world"}`)))
	})

	It("Should correctly audit the request with logfile group and mode set", func() {
		if os.Getuid() != 0 {
			Skip("File mode test only ran as root user")
		}

		tf := cfg.Choria.RPCAuditLogfile
		cfg, err = config.NewConfig("testdata/audit-group-mode.cfg")
		Expect(err).ToNot(HaveOccurred())
		cfg.Choria.RPCAuditLogfile = tf

		Expect(cfg.RPCAudit).To(BeTrue())
		Expect(cfg.Choria.RPCAuditLogfile).ToNot(BeAnExistingFile())

		req, err := v1.NewRequest("test_agent", "test.node", "choria=rip.mcollective", 120, "uniq_req_id", "mcollective")
		Expect(err).ToNot(HaveOccurred())

		ok := Request(req, "test_agent", "test_action", json.RawMessage(`{"hello":"world"}`), cfg)
		Expect(ok).To(BeTrue())
		Expect(cfg.Choria.RPCAuditLogfile).To(BeAnExistingFile())

		j, err := os.ReadFile(cfg.Choria.RPCAuditLogfile)
		Expect(err).ToNot(HaveOccurred())

		am := Message{}
		err = json.Unmarshal(j, &am)
		Expect(err).ToNot(HaveOccurred())

		Expect(am.RequestID).To(Equal(req.RequestID()))
		Expect(am.RequestTime).To(Equal(req.Time().UTC().Unix()))
		Expect(am.CallerID).To(Equal("choria=rip.mcollective"))
		Expect(am.Sender).To(Equal("test.node"))
		Expect(am.Agent).To(Equal("test_agent"))
		Expect(am.Action).To(Equal("test_action"))
		Expect(am.Data).To(Equal(json.RawMessage(`{"hello":"world"}`)))

		if runtime.GOOS != "windows" {
			stat, err := os.Stat(cfg.Choria.RPCAuditLogfile)
			Expect(err).ToNot(HaveOccurred())

			unixPerms := stat.Mode() & os.ModePerm
			permString := fmt.Sprintf("%v", unixPerms)
			Expect(permString).To(Equal("-rw-r-----"))

			group := cfg.Choria.RPCAuditLogfileGroup
			checkFileGid(stat, group)
		}
	})
})
