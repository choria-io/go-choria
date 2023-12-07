// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/aagent/watchers/pluginswatcher"
	"github.com/choria-io/go-choria/config"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"
)

type mPluginsAddCommand struct {
	command

	matcher        string
	matcherIsSet   bool
	governor       string
	governorIsSet  bool
	pluginUrl      *url.URL
	manifest       string
	checksum       string
	verifyChecksum string
	name           string
}

func (c *mPluginsAddCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine plugins"); ok {
		c.cmd = machine.Cmd().Command("add", "Add or Update a reference to a plugin to the Plugins Watcher manifest")
		c.cmd.Arg("name", "Unique name for this plugin").Required().StringVar(&c.name)
		c.cmd.Arg("plugin", "The URL to the tar file holding the plugin").Required().URLVar(&c.pluginUrl)
		c.cmd.Arg("checksum", "SHA256 checksum of the archive").Required().StringVar(&c.checksum)
		c.cmd.Arg("verify-checksum", "SHA256 checksum of the verification file").Required().StringVar(&c.verifyChecksum)
		c.cmd.Flag("manifest", "Path to the manifest to edit").Default("plugins.json").StringVar(&c.manifest)
		c.cmd.Flag("matcher", "Limit deployment using an expression").IsSetByUser(&c.matcherIsSet).StringVar(&c.matcher)
		c.cmd.Flag("governor", "Limit deployment using a Governor").IsSetByUser(&c.governorIsSet).StringVar(&c.governor)
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &mPluginsAddCommand{})
}

func (c *mPluginsAddCommand) Configure() error {
	if debug {
		logrus.SetOutput(os.Stdout)
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Logging at debug level due to CLI override")
	}

	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return err
	}

	cfg.Choria.SecurityProvider = "file"
	cfg.DisableSecurityProviderVerify = true

	return err
}

func (c *mPluginsAddCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	var manifest []*pluginswatcher.ManagedPlugin

	if iu.FileExist(c.manifest) {
		dat, err := os.ReadFile(c.manifest)
		if err != nil {
			return err
		}

		err = json.Unmarshal(dat, &manifest)
		if err != nil {
			return err
		}
	}

	user := c.pluginUrl.User
	c.pluginUrl.User = nil
	var found bool

	for _, p := range manifest {
		if p.Name == c.name {
			c.updatePlugin(p, user)
			found = true

			fmt.Printf("Updating plugin %s\n", p.Name)
		}
	}

	if !found {
		p := &pluginswatcher.ManagedPlugin{}
		c.updatePlugin(p, user)

		name := strings.Split(c.pluginUrl.Path[1:], "-")
		if len(name) == 0 {
			return fmt.Errorf("could not determine name, please pass --name")
		}
		p.Name = name[0]

		fmt.Printf("Adding plugin %s\n", p.Name)
		manifest = append(manifest, p)
	}

	dat, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(c.manifest, dat, 0600)
	if err != nil {
		return err
	}

	fmt.Printf("Use the pack command to sign the %s manifest\n", c.manifest)

	return nil
}

func (c *mPluginsAddCommand) updatePlugin(plugin *pluginswatcher.ManagedPlugin, user *url.Userinfo) {
	plugin.Source = c.pluginUrl.String()
	plugin.ArchiveChecksum = c.checksum
	plugin.ContentChecksumsChecksum = c.verifyChecksum

	if c.governorIsSet {
		plugin.Governor = c.governor
	}
	if c.matcherIsSet {
		plugin.Matcher = c.matcher
	}
	if user != nil {
		plugin.Username = user.Username()
		plugin.Password, _ = user.Password()
	}

	if c.name != "" {
		plugin.Name = c.name
	}
}
