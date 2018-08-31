package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	"github.com/choria-io/go-protocol/protocol"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
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
var cfg *config.Config
var ctx context.Context
var cancel func()
var wg *sync.WaitGroup
var mu = &sync.Mutex{}
var err error
var profile string

func ParseCLI() (err error) {
	cli.app = kingpin.New("choria", "Choria Orchestration System")
	cli.app.Version(build.Version)
	cli.app.Author("R.I.Pienaar <rip@devco.net>")

	cli.app.Flag("debug", "Enable debug logging").Short('d').BoolVar(&debug)
	cli.app.Flag("config", "Config file to use").StringVar(&configFile)
	cli.app.Flag("profile", "Enable CPU profiling and write to the supplied file").Hidden().StringVar(&profile)

	for _, cmd := range cli.commands {
		err = cmd.Setup()
	}

	cli.command = kingpin.MustParse(cli.app.Parse(os.Args[1:]))

	for _, cmd := range cli.commands {
		if cmd.FullCommand() == cli.command {
			err = cmd.Configure()
			if err != nil {
				return fmt.Errorf("%s failed to configure: %s", cmd.FullCommand(), err)
			}
		}
	}

	return
}

func commonConfigure() error {
	if debug {
		log.SetOutput(os.Stdout)
		log.SetLevel(log.DebugLevel)
		log.Debug("Logging at debug level due to CLI override")
	}

	if configFile == "" {
		configFile = choria.UserConfig()
	}

	cfg, err = config.NewConfig(configFile)
	if err != nil {
		return fmt.Errorf("Could not parse configuration: %s", err)
	}

	if os.Getenv("INSECURE_YES_REALLY") == "true" {
		protocol.Secure = "false"
		cfg.DisableTLS = true
	}

	return nil
}

func Run() (err error) {
	wg = &sync.WaitGroup{}
	ran := false

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	go interruptWatcher()

	if profile != "" {
		f, err := os.Create(profile)
		if err != nil {
			return fmt.Errorf("could not setup profiling: %s", err)
		}

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// we do this here so that the command setup has a chance to fiddle the config for
	// things like disabling full verification of the security system during enrollment
	c, err = choria.NewWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("Could not initialize Choria: %s", err)
	}

	c.SetupLogging(debug)

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
