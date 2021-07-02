package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/governor"
)

type tGovRunCommand struct {
	command
	name     string
	fullCmd  string
	maxWait  time.Duration
	interval time.Duration
}

func (g *tGovRunCommand) Setup() (err error) {
	if gen, ok := cmdWithFullCommand("tool governor"); ok {
		g.cmd = gen.Cmd().Command("run", "Runs a command subject to Governor control")
		g.cmd.Arg("name", "The name for the Governor").Required().StringVar(&g.name)
		g.cmd.Arg("command", "Command to execute").Required().StringVar(&g.fullCmd)
		g.cmd.Flag("max-wait", "Maximum amount of time to wait to obtain a lease").Default("5m").DurationVar(&g.maxWait)
		g.cmd.Flag("interval", "Interval for attempting to get a lease").Default("5s").DurationVar(&g.interval)
	}

	return nil
}

func (g *tGovRunCommand) Configure() (err error) {
	// we pretend to be a server so that we use the right certs and identity etc
	return systemConfigureIfRoot(true)
}

func (g *tGovRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if g.interval < time.Second {
		return fmt.Errorf("interval should be >=1s")
	}

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("governor manager: %s", g.name), c.Logger("governor"))
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	subj := fmt.Sprintf("%s.governor.%s", cfg.MainCollective, g.name)
	gov := governor.NewJSGovernor(g.name, mgr, governor.WithSubject(subj), governor.WithInterval(g.interval))

	parts, err := shellquote.Split(g.fullCmd)
	if err != nil {
		return fmt.Errorf("can not parse command: %s", err)
	}
	var cmd string
	var args []string

	switch {
	case len(parts) == 0:
		return fmt.Errorf("could not parse command")
	case len(parts) == 1:
		cmd = parts[0]
	default:
		cmd = parts[0]
		args = append(args, parts[1:]...)
	}

	ctx, cancel := context.WithTimeout(ctx, g.maxWait)
	defer cancel()

	finisher, err := gov.Start(ctx, cfg.Identity)
	if err != nil {
		return fmt.Errorf("could not get a execution slot: %s", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		finisher()
	}()

	osExit := func(c int, format string, a ...interface{}) {
		finisher()

		if format != "" {
			fmt.Println(fmt.Sprintf(format, a...))
		}

		os.Exit(c)
	}

	execution := exec.Command(cmd, args...)
	execution.Stdin = os.Stdin
	execution.Stdout = os.Stdout
	execution.Stderr = os.Stderr
	err = execution.Run()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			osExit(exitErr.ExitCode(), "")
		} else {
			osExit(1, "execution failed: %s", err)
		}
	}

	if execution.ProcessState == nil {
		osExit(1, "Unknown execution state")
	}

	osExit(execution.ProcessState.ExitCode(), "")

	return nil

}

func init() {
	cli.commands = append(cli.commands, &tGovRunCommand{})
}
