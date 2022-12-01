// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/choria-io/go-choria/build"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChoria(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config")
}

var _ = Describe("Choria/Config", func() {
	Describe("NewConfig", func() {
		AfterEach(func() {
			build.DefaultCollectives = "mcollective"
		})

		It("Should switch to choria collective when not configured and the choria security system is in use", func() {
			c := &Config{Choria: &ChoriaPluginConfig{SecurityProvider: "choria"}}
			err := c.normalize()
			Expect(err).ToNot(HaveOccurred())
			Expect(c.Collectives).To(Equal([]string{"choria"}))

			c = &Config{Choria: &ChoriaPluginConfig{SecurityProvider: "puppet"}}
			err = c.normalize()
			Expect(err).ToNot(HaveOccurred())
			Expect(c.Collectives).To(Equal([]string{"mcollective"}))

			build.DefaultCollectives = "foo"
			c = &Config{Choria: &ChoriaPluginConfig{SecurityProvider: "puppet"}}
			err = c.normalize()
			Expect(err).ToNot(HaveOccurred())
			Expect(c.Collectives).To(Equal([]string{"foo"}))

			c = &Config{Choria: &ChoriaPluginConfig{}, Collectives: []string{"x", "y"}}
			err = c.normalize()
			Expect(err).ToNot(HaveOccurred())
			Expect(c.Collectives).To(Equal([]string{"x", "y"}))
			Expect(c.MainCollective).To(Equal("x"))
		})

		It("Should get collectives from build settings", func() {
			c := &Config{Choria: &ChoriaPluginConfig{}}
			build.DefaultCollectives = "g1 , g2"
			err := c.normalize()
			Expect(err).To(Not(HaveOccurred()))
			Expect(c.Collectives).To(Equal([]string{"g1", "g2"}))
			Expect(c.MainCollective).To(Equal("g1"))
		})

		It("Should correctly parse config files", func() {
			var c *Config
			var err error

			forceDotParse = true
			if runtime.GOOS == "windows" {
				c, err = NewConfig("testdata/choria_windows.cfg")
			} else {
				c, err = NewConfig("testdata/choria.cfg")
			}
			Expect(err).ToNot(HaveOccurred())
			forceDotParse = false

			Expect(c.Choria.NetworkWriteDeadline).To(Equal(10 * time.Second))
			Expect(c.Registration).To(Equal([]string{"foo"}))
			Expect(c.RegisterInterval).To(Equal(10))
			Expect(c.RegistrationSplay).To(BeTrue())
			Expect(c.Collectives).To(Equal([]string{"c_1", "c_2", "c_3"}))
			Expect(c.MainCollective).To(Equal("c_1"))
			Expect(c.LibDir).To(Equal([]string{"/dir1", "/dir2", "/dir3", "/dir4"}))
			Expect(c.DefaultDiscoveryOptions).To(Equal([]string{"one", "two"}))

			if runtime.GOOS == "windows" {
				Expect(c.Color).To(BeFalse())
			} else {
				Expect(c.Color).To(BeTrue())
			}

			Expect(c.Choria.PrivilegedUsers).To(Equal([]string{
				"\\.privileged.mcollective$",
				"\\.privileged.choria$",
			}))
			Expect(c.Choria.CertnameAllowList).To(Equal([]string{
				"\\.mcollective$",
				"\\.choria$",
			}))

			Expect(c.Option("plugin.package.setting", "default")).To(Equal("1"))
			Expect(c.Option("plugin.package.other_setting", "default")).To(Equal("default"))

			c.SetOption("plugin.package.other_setting", "override")
			Expect(c.Option("plugin.package.setting", "default")).To(Equal("1"))
			Expect(c.Option("plugin.package.other_setting", "default")).To(Equal("override"))
		})
	})

	Context("Projects", func() {
		It("Should find the project configs", func() {
			c, err := ProjectConfigurationFiles("testdata/project")
			Expect(err).ToNot(HaveOccurred())
			pwd, _ := os.Getwd()
			Expect(len(c)).To(BeNumerically(">=", 1))
			Expect(c[len(c)-1]).To(Equal(filepath.Join(pwd, "testdata", "project", "choria.conf")))
		})

		It("Should load project configs for users", func() {
			pwd, _ := os.Getwd()
			Expect(os.Chdir(filepath.Join(pwd, "testdata", "project"))).ToNot(HaveOccurred())
			defer os.Chdir(pwd)

			cfg, err := NewConfig(filepath.Join("..", "choria.cfg"))
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Option("plugin.project.test", "")).To(Equal("1"))
		})

		It("Should not load project configs for system components", func() {
			pwd, _ := os.Getwd()
			Expect(os.Chdir(filepath.Join(pwd, "testdata", "project"))).ToNot(HaveOccurred())
			defer os.Chdir(pwd)

			cfg, err := NewSystemConfig(filepath.Join("..", "choria.cfg"), false)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Option("plugin.project.test", "")).To(Equal("0"))
		})
	})
})
