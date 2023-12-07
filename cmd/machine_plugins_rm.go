// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/choria-io/go-choria/aagent/watchers/pluginswatcher"
	"github.com/choria-io/go-choria/config"
	"github.com/sirupsen/logrus"
)

type mPluginsRmCommand struct {
	command

	name     string
	manifest string
}

func (c *mPluginsRmCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine plugins"); ok {
		c.cmd = machine.Cmd().Command("rm", "Removes references to a plugin from the Plugins Watcher manifest")
		c.cmd.Arg("name", "Plugin name to remove from the manifest").Required().StringVar(&c.name)
		c.cmd.Flag("manifest", "Path to the manifest to edit").Default("plugins.json").ExistingFileVar(&c.manifest)
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &mPluginsRmCommand{})
}

func (c *mPluginsRmCommand) Configure() error {
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

func (c *mPluginsRmCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	var manifest []*pluginswatcher.ManagedPlugin

	dat, err := os.ReadFile(c.manifest)
	if err != nil {
		return err
	}

	err = json.Unmarshal(dat, &manifest)
	if err != nil {
		return err
	}

	var found bool
	newManifest := []*pluginswatcher.ManagedPlugin{}

	for _, p := range manifest {
		if p.Name == c.name {
			fmt.Printf("Removing plugin %s\n", c.name)
			found = true
			continue
		}

		newManifest = append(newManifest, p)
	}

	if !found {
		return fmt.Errorf("no plugin named %s found", c.name)
	}

	dat, err = json.MarshalIndent(newManifest, "", "  ")
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
