package cmd

import (
	"sync"
)

type serverCommand struct {
	command
}

type serverRunCommand struct {
	command

	serviceHost      bool
	disableTLS       bool
	disableTLSVerify bool
	pidFile          string
}

// server
func (b *serverCommand) Setup() (err error) {
	b.cmd = cli.app.Command("server", "Choria Server")

	return
}

func (b *serverCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return
}

func (b *serverCommand) Configure() error {
	cfg.DisableSecurityProviderVerify = true

	return nil
}

func init() {
	cli.commands = append(cli.commands, &serverCommand{})
}
