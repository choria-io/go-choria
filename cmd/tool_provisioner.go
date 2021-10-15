// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/config"
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
	err := commonConfigure()
	if err != nil {
		return err
	}

	cfg, err = config.NewDefaultSystemConfig(true)
	if err != nil {
		return err
	}

	cfg.ApplyBuildSettings(bi)

	cfg.DisableSecurityProviderVerify = true
	cfg.InitiatedByServer = true
	cfg.Choria.Provision = true

	return nil
}

func (p *tProvisionerCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	c.ConfigureProvisioning()

	if !c.ProvisionMode() {
		return fmt.Errorf("not a server compiled for auto provisioning or no JWT token found to enable it")
	}

	fmt.Printf("Attempting provisioner resolution using: %s\n", provtarget.Name())

	targets, err := provtarget.Targets(ctx, c.Logger("provisioner"))
	if err != nil {
		return err
	}

	fmt.Printf("Provisioning using %d broker(s):\n\n", targets.Count())
	fmt.Print(strings.Join(targets.Strings(), "\t"))

	fmt.Println()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tProvisionerCommand{})
}
