package cmd

import (
	"fmt"
	"sync"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
)

type enrollCommand struct {
	command

	cn string
}

func (e *enrollCommand) Setup() (err error) {
	e.cmd = cli.app.Command("enroll", "Enrolls this node with the security provider")
	e.cmd.Flag("certname", "Custom Certificate Name").StringVar(&e.cn)

	return
}

func (e *enrollCommand) Configure() error {
	err = commonConfigure()
	if err != nil {
		return err
	}

	cfg.DisableSecurityProviderVerify = true

	if e.cn != "" {
		cfg.OverrideCertname = e.cn
	}

	return nil
}

func (e *enrollCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	fmt.Printf("Enrolling with the Security System using certname %s\n", c.Certname())

	err = c.Enroll(ctx, 250*time.Second, func(try int) { fmt.Printf("Attempting to download certificate for %s, try %d.\n", c.Certname(), try) })
	if err != nil {
		kingpin.Errorf("Could not enroll: %s", err)
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &enrollCommand{})
}
