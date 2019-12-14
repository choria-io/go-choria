package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/provtarget"
	"github.com/choria-io/go-config"
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

	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return err
	}

	cfg.ApplyBuildSettings(&build.Info{})

	cfg.DisableSecurityProviderVerify = true
	cfg.InitiatedByServer = true
	cfg.Choria.Provision = true

	return nil
}

func (p *tProvisionerCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

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
