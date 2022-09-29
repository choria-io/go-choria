// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/providers/provtarget"
)

type tProvisionerCommand struct {
	command
}

func (p *tProvisionerCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		p.cmd = tool.Cmd().Command("provisioner", "View the provisioner targets based on related plugins")
	}

	return nil
}

func (p *tProvisionerCommand) Configure() error {
	if debug {
		log.SetOutput(os.Stdout)
		log.SetLevel(log.DebugLevel)
		log.Debug("Logging at debug level due to CLI override")
	}

	if configFile == "" {
		return fmt.Errorf("please specify the server configuration using --config")
	}

	if util.FileExist(configFile) {
		cfg, err = config.NewSystemConfig(configFile, true)
		if err != nil {
			return err
		}
	} else {
		log.Warnf("Using generated default config as %s does not exist", configFile)
		cfg, err = config.NewDefaultSystemConfig(true)
		cfg.Choria.SecurityProvider = "file"
		cfg.Choria.SSLDir = "/tmp"
	}

	cfg.LogFile = ""
	cfg.LoggerType = "console"
	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.Provision = true
	if debug {
		cfg.LogLevel = "debug"
	}

	cfg.ApplyBuildSettings(bi)

	return nil
}

func (p *tProvisionerCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	c.ConfigureProvisioning()

	if !c.ProvisionMode() {
		return fmt.Errorf("not a server compiled for auto provisioning or the provisioning target is not functional")
	}

	fmt.Printf("Attempting provisioner resolution using: %s\n", provtarget.Name())

	targets, err := provtarget.Targets(ctx, c.Logger("provisioner"))
	if err != nil {
		return err
	}

	fmt.Printf("Provisioning using %d broker(s):\n\n", targets.Count())
	for _, t := range targets.Strings() {
		fmt.Printf("\t%s\n", t)
	}

	fmt.Println()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tProvisionerCommand{})
}
