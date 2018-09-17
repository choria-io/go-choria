package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/provtarget"
)

type provisionerCommand struct {
	command
}

func (p *provisionerCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		p.cmd = tool.Cmd().Command("provisioner", "View the provisioner targets based on related plugins")
	}

	return nil
}

func (p *provisionerCommand) Configure() error {
	err := commonConfigure()
	if err != nil {
		return err
	}

	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return err
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.InitiatedByServer = true
	cfg.Choria.Provision = true

	return nil
}

func (p *provisionerCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if !c.ProvisionMode() {
		return fmt.Errorf("not a server compiled for auto provisioning")
	}

	fmt.Printf("Attempting provisioner resolution using: %s\n", provtarget.Name())

	targets, err := provtarget.Targets(ctx, c.Logger("provisioner"))
	if err != nil {
		return err
	}

	fmt.Printf("Provisioning using %d broker(s):\n\n", len(targets))

	for _, t := range targets {
		fmt.Printf("\t%s", t.String())
	}

	fmt.Println()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &provisionerCommand{})
}
