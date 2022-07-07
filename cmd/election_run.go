// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	iu "github.com/choria-io/go-choria/internal/util"
	election "github.com/choria-io/go-choria/providers/election/streams"
	"github.com/kballard/go-shellquote"
	"github.com/sirupsen/logrus"
)

type tElectRunCommand struct {
	command
	fullCmd    []string
	name       string
	bucket     string
	killOnLost bool
	proc       *exec.Cmd
	state      election.State
	executable string
	args       []string
	signaled   bool

	log *logrus.Entry
	mu  sync.Mutex
}

func (f *tElectRunCommand) Setup() (err error) {
	if elect, ok := cmdWithFullCommand("election"); ok {
		f.cmd = elect.Cmd().Command("run", "Runs a command under leader election control")
		f.cmd.Arg("name", "The name for the Leader Election to campaign in").Required().StringVar(&f.name)
		f.cmd.Arg("command", "Command to execute").Required().StringsVar(&f.fullCmd)
		f.cmd.Flag("terminate", "Terminates the command when leadership is lost").UnNegatableBoolVar(&f.killOnLost)
		f.cmd.Flag("bucket", "Use a specific bucket for elections").Default("CHORIA_LEADER_ELECTION").StringVar(&f.bucket)
	}

	return nil
}

func (f *tElectRunCommand) Configure() (err error) {
	parts, err := shellquote.Split(strings.Join(f.fullCmd, " "))
	if err != nil {
		return fmt.Errorf("can not parse command: %s", err)
	}

	switch {
	case len(parts) == 0:
		return fmt.Errorf("could not parse command")
	case len(parts) == 1:
		f.executable = parts[0]
	default:
		f.executable = parts[0]
		f.args = append(f.args, parts[1:]...)
	}

	return commonConfigure()
}

func (f *tElectRunCommand) handleLeaderState() {
	f.mu.Lock()
	f.state = election.LeaderState
	proc := f.proc
	signaled := f.signaled
	f.mu.Unlock()

	if proc != nil {
		if !signaled {
			p := f.proc.Process
			if p != nil {
				f.log.Infof("Sending USR1 to %d", p.Pid)
				p.Signal(syscall.SIGUSR1)
				f.mu.Lock()
				f.signaled = true
				f.mu.Unlock()
			}
		}
	} else {
		go f.runCommand()
	}
}

func (f *tElectRunCommand) handleCampaignerState() {
	f.mu.Lock()
	defer f.mu.Unlock()

	lost := false
	if f.state == election.LeaderState {
		f.log.Infof("Leadership lost")
		f.state = election.CandidateState
		lost = true
		f.signaled = false
	}

	if f.proc == nil || !lost {
		return
	}

	process := f.proc.Process
	if process == nil {
		return
	}

	if f.killOnLost {
		if runtime.GOOS != "windows" {
			f.log.Warnf("Sending SIGINT to %d", process.Pid)
			process.Signal(syscall.SIGINT)

			if iu.InterruptibleSleep(ctx, time.Second) == context.Canceled {
				return
			}
		}
		f.log.Warnf("Sending SIGTERM to %d", process.Pid)
		process.Signal(syscall.SIGTERM)
	} else {
		f.log.Warnf("Sending SIGUSR2 to %d", process.Pid)
		process.Signal(syscall.SIGUSR2)
	}
}

func (f *tElectRunCommand) campaign(s election.State) {
	switch s {
	case election.LeaderState:
		f.handleLeaderState()
	default:
		f.handleCampaignerState()
	}
}

func (f *tElectRunCommand) won() {
	f.log.Infof("Became leader")
}

func (f *tElectRunCommand) lost() {
	f.handleCampaignerState()
}

func (f *tElectRunCommand) runCommand() {
	f.log.Infof("Running command %q with %s", f.executable, f.args)
	f.proc = exec.Command(f.executable, f.args...)
	f.proc.Stdin = os.Stdin
	f.proc.Stdout = os.Stdout
	f.proc.Stderr = os.Stderr
	err = f.proc.Start()
	if err != nil {
		f.log.Errorf("Execution failed: %v", err)
		os.Exit(1)
	}

	// give it some time to start properly
	if iu.InterruptibleSleep(ctx, time.Second) == context.Canceled {
		f.log.Errorf("Exiting on context interrupt")
		os.Exit(1)
	}

	if f.proc.Process != nil {
		f.mu.Lock()
		f.proc.Process.Signal(syscall.SIGUSR1)
		f.signaled = true
		f.mu.Unlock()
	}

	err = f.proc.Wait()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			code := exitErr.ExitCode()
			if code == -1 {
				code = 1
			}
			f.log.Errorf("Execution failed with exit code: %d", code)
			os.Exit(code)
		} else {
			f.log.Errorf("Execution failed: %v", err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}

func (f *tElectRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	f.log = c.Logger("election")

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("election %s %s", f.name, c.Config.Identity), f.log)
	if err != nil {
		return err
	}

	js, err := conn.Nats().JetStream()
	if err != nil {
		return err
	}

	kv, err := js.KeyValue(f.bucket)
	if err != nil {
		return fmt.Errorf("cannot access KV Bucket %s: %v", f.bucket, err)
	}

	el, err := election.NewElection(c.Config.Identity, f.name, kv, election.OnLost(f.lost), election.OnCampaign(f.campaign), election.OnWon(f.won), election.WithDebug(f.log.Infof))
	if err != nil {
		return err
	}

	return el.Start(ctx)
}

func init() {
	cli.commands = append(cli.commands, &tElectRunCommand{})
}
