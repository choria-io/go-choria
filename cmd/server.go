package cmd

import (
	"os"
	"sync"

	"github.com/choria-io/go-choria/server"
	log "github.com/sirupsen/logrus"
)

type serverCommand struct {
	command
}

type serverRunCommand struct {
	command

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
	return
}

// server run
func (r *serverRunCommand) Setup() (err error) {
	if broker, ok := cmdWithFullCommand("server"); ok {
		r.cmd = broker.Cmd().Command("run", "Runs a Choria Server").Default()
		r.cmd.Flag("disable-tls", "Disables TLS").Hidden().Default("false").BoolVar(&r.disableTLS)
		r.cmd.Flag("disable-ssl-verification", "Disables SSL Verification").Hidden().Default("false").BoolVar(&r.disableTLSVerify)
		r.cmd.Flag("pid", "Write running PID to a file").StringVar(&r.pidFile)
	}

	return
}

func (r *serverRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	instance, err := server.NewInstance(c)
	if err != nil {
		log.Errorf("Could not start choria: %s", err.Error())
		os.Exit(1)
	}

	wg.Add(1)
	instance.Run(ctx, wg)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &serverCommand{})
	cli.commands = append(cli.commands, &serverRunCommand{})
}
