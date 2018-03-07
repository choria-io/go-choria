package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

type application struct {
	app      *kingpin.Application
	command  string
	commands []runableCmd
}

var cli = application{}
var debug = false
var configFile = ""
var c *choria.Framework
var config *choria.Config
var ctx context.Context
var cancel func()
var wg *sync.WaitGroup
var mu = &sync.Mutex{}

func ParseCLI() (err error) {
	cli.app = kingpin.New("choria", "Choria Orchestration System")
	cli.app.Version(build.Version)
	cli.app.Author("R.I.Pienaar <rip@devco.net>")
	cli.app.Flag("debug", "Enable debug logging").Short('d').BoolVar(&debug)
	cli.app.Flag("config", "Config file to use").StringVar(&configFile)

	for _, cmd := range cli.commands {
		err = cmd.Setup()
	}

	cli.command = kingpin.MustParse(cli.app.Parse(os.Args[1:]))

	// skip initialization for buildinfo, people might want to see this
	// even if their SSL is invalid etc
	if cli.command == "buildinfo" {
		return
	}

	if debug {
		log.SetOutput(os.Stdout)
		log.SetLevel(log.DebugLevel)
		log.Debug("Logging at debug level due to CLI override")
	}

	if configFile == "" {
		configFile = choria.UserConfig()
	}

	if c, err = choria.New(configFile); err != nil {
		return fmt.Errorf("Could not initialize Choria: %s", err)
	}

	config = c.Config

	c.SetupLogging(debug)

	return
}

func Run() (err error) {
	wg = &sync.WaitGroup{}
	ran := false

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	go interruptWatcher()

	for _, cmd := range cli.commands {
		if cmd.FullCommand() == cli.command {
			ran = true

			wg.Add(1)
			err = cmd.Run(wg)
		}
	}

	if !ran {
		err = fmt.Errorf("Could not run the CLI: Invalid command %s", cli.command)
	}

	if err != nil {
		log.Errorf("Shutting down due to: %s", err)
		cancel()
	}

	wg.Wait()

	return
}

func interruptWatcher() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				log.Infof("Shutting down on %s", sig)
				cancel()
			case syscall.SIGQUIT:
				dumpGoRoutines()
			}
		case <-ctx.Done():
			return
		}
	}
}

func dumpGoRoutines() {
	mu.Lock()
	defer mu.Unlock()

	outname := filepath.Join(os.TempDir(), fmt.Sprintf("choria-threaddump-%d-%d.txt", os.Getpid(), time.Now().UnixNano()))

	buf := make([]byte, 1<<20)
	stacklen := runtime.Stack(buf, true)

	err := ioutil.WriteFile(outname, buf[:stacklen], 0644)
	if err != nil {
		log.Errorf("Could not produce thread dump: %s", err)
		return
	}

	log.Warnf("Produced thread dump to %s", outname)
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
