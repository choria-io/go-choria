package cmd

import (
	"fmt"
	"os"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

type application struct {
	app      *kingpin.Application
	command  string
	commands []runableCmd
}

const version = "0.0.1"

var cli = application{}
var debug = false
var configFile = ""
var c *choria.Framework

func ParseCLI() (err error) {
	cli.app = kingpin.New("choria", "Choria Orchestration System")
	cli.app.Version(version)
	cli.app.Author("R.I.Pienaar <rip@devco.net>")
	cli.app.Flag("debug", "Enable debug logging").Short('d').BoolVar(&debug)
	cli.app.Flag("config", "Config file to use").StringVar(&configFile)

	for _, cmd := range cli.commands {
		err = cmd.Setup()
	}

	cli.command = kingpin.MustParse(cli.app.Parse(os.Args[1:]))

	if debug {
		log.SetOutput(os.Stdout)
		log.SetLevel(log.DebugLevel)
		log.Debug("Logging at debug level due to CLI override")
	}

	if configFile == "" {
		configFile = choria.UserConfig()
	}

	if c, err = choria.New(configFile); err != nil {
		return fmt.Errorf("Could not initialize Choria: %s", err.Error())
	}

	return
}

func Run() (err error) {
	ran := false

	for _, cmd := range cli.commands {
		if cmd.FullCommand() == cli.command {
			ran = true
			err = cmd.Run()
		}
	}

	if !ran {
		err = fmt.Errorf("Could not run the CLI: Invalid command %s", cli.command)
	}

	return
}

// digs in the application.commands structure looking for a entry with
// the given command string
func cmdWithFullCommand(command string) (cmd runableCmd, ok bool) {
	for _, cmd := range cli.commands {
		if cmd.FullCommand() == command {
			return cmd, true
		}
	}

	return cmd, false
}
