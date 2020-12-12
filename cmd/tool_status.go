package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/statistics"
)

type tStatusCommand struct {
	command
	statusFile     string
	checkConnected bool
	lastMessage    time.Duration
	maxAge         time.Duration
}

func (s *tStatusCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		s.cmd = tool.Cmd().Command("status", "Checks the health of a running Choria instance based on its status file")
		s.cmd.Flag("status-file", "The status file to check").Required().ExistingFileVar(&s.statusFile)
		s.cmd.Flag("disconnected", "Checks if the server is connected to a broker").Default("true").BoolVar(&s.checkConnected)
		s.cmd.Flag("message-since", "Maximum time to allow no messages to pass (0 disables)").Default("1h").DurationVar(&s.lastMessage)
		s.cmd.Flag("max-age", "Maximum age for the status file (0 disables)").Default("30m").DurationVar(&s.maxAge)
	}

	return nil
}

func (s *tStatusCommand) Configure() error {
	return nil
}

func (s *tStatusCommand) checkConnection(status *statistics.InstanceStatus) (err error) {
	if !s.checkConnected {
		return nil
	}

	return status.CheckConnection()
}

func (s *tStatusCommand) checkLastMessage(status *statistics.InstanceStatus) (err error) {
	if s.lastMessage == 0 {
		return nil
	}

	return status.CheckLastMessage(s.lastMessage)
}

func (s *tStatusCommand) checkFileAge(status *statistics.InstanceStatus) (err error) {
	if s.maxAge == 0 {
		return nil
	}

	return status.CheckFileAge(s.maxAge)
}

func (s *tStatusCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	status, err := statistics.LoadInstanceStatus(s.statusFile)
	if err != nil {
		s.exit(fmt.Errorf("%s could not be read: %s", s.statusFile, err))
	}

	err = s.checkFileAge(status)
	if err != nil {
		s.exit(err)
	}

	err = s.checkConnection(status)
	if err != nil {
		s.exit(fmt.Errorf("connection check failed: %s", err))
	}

	err = s.checkLastMessage(status)
	if err != nil {
		s.exit(fmt.Errorf("no recent messages: %s", err))
	}

	s.exit(nil)

	return nil
}

func (s *tStatusCommand) exit(err error) {
	if err != nil {
		fmt.Printf("%s %s\n", s.statusFile, err)
		os.Exit(1)
	}

	fmt.Printf("%s OK\n", s.statusFile)
	os.Exit(0)
}

func init() {
	cli.commands = append(cli.commands, &tStatusCommand{})
}
