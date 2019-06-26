package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-config"
	lifecycle "github.com/choria-io/go-lifecycle"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	// Version is the release version to be set at compile time
	Version = "development"

	viewcmd  *kingpin.CmdClause
	tallycmd *kingpin.CmdClause

	port            int
	debug           bool
	tls             bool
	componentFilter string
	typeFilter      string
	pidfile         string
	cfgfile         string
	prefix          string
	fw              *choria.Framework

	ctx    context.Context
	cancel func()
)

func Run() {
	app := kingpin.New("lifecycle", "The Choria lifecycle event manager")
	app.Author("R.I.Pienaar <rip@devco.net>")
	app.Version(Version)
	app.Flag("config", "Configuration file to use").ExistingFileVar(&cfgfile)
	app.Flag("debug", "Enable debug logging").BoolVar(&debug)
	app.Flag("tls", "Use TLS when connecting to the middleware").Default("true").BoolVar(&tls)

	viewcmd = app.Command("view", "View real time lifecycle events")
	viewcmd.Flag("component", "Limit events to a named component").StringVar(&componentFilter)
	viewcmd.Flag("type", "Limits the events to a particular type").EnumVar(&typeFilter, lifecycle.EventTypeNames()...)

	tallycmd = app.Command("tally", "Record lifecycle events and report metrics to Prometheus")
	tallycmd.Flag("component", "Component to tally").StringVar(&componentFilter)
	tallycmd.Flag("port", "Port to listen on").Default("8080").IntVar(&port)
	tallycmd.Flag("prefix", "Prefix for statistic keys").StringVar(&prefix)

	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	go interruptWatcher()

	if pidfile != "" {
		err := ioutil.WriteFile(pidfile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
		kingpin.FatalIfError(err, "Could not write pid file %s: %s", pidfile, err)
	}

	if cfgfile == "" {
		cfgfile = choria.UserConfig()
	}

	cfg, err := config.NewConfig(cfgfile)
	kingpin.FatalIfError(err, "could not parse configuration: %s", err)

	if debug {
		cfg.LogLevel = "debug"
		cfg.LogFile = ""
	}

	if !tls {
		cfg.DisableTLS = true
	}

	fw, err = choria.NewWithConfig(cfg)
	kingpin.FatalIfError(err, "could not set up choria: %s", err)

	switch command {
	case viewcmd.FullCommand():
		err = view()
	case tallycmd.FullCommand():
		err = recordtally()
	}

	kingpin.FatalIfError(err, "Could not run %s: %s", command, err)
}

func interruptWatcher() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				cancel()
			}
		case <-ctx.Done():
			return
		}
	}
}
