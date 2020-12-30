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

	err = c.Enroll(ctx, 250*time.Second, func(digest string, try int) {
		if digest != "" && try <= 1 {
			fmt.Printf("Certificate fingerprint: %s\n\n", digest)
		}
		fmt.Printf("Attempting to download certificate for %s, try %d\n", c.Certname(), try)
	})
	kingpin.FatalIfError(err, "Could not enroll")

	return nil
}

func init() {
	cli.commands = append(cli.commands, &enrollCommand{})
}
