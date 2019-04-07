package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/server"
)

type statusCommand struct {
	command
	statusFile     string
	checkConnected bool
	lastMessage    time.Duration
	maxAge         time.Duration
}

func (s *statusCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		s.cmd = tool.Cmd().Command("status", "Checks the health of a running Choria instance based on its status file")
		s.cmd.Flag("status-file", "The status file to check").Required().ExistingFileVar(&s.statusFile)
		s.cmd.Flag("disconnected", "Checks if the server is connected to a broker").Default("true").BoolVar(&s.checkConnected)
		s.cmd.Flag("message-since", "Maximum time to allow no messages to pass (0 disables)").Default("1h").DurationVar(&s.lastMessage)
		s.cmd.Flag("max-age", "Maximum age for the status file (0 disables)").Default("30m").DurationVar(&s.maxAge)
	}

	return nil
}

func (s *statusCommand) Configure() error {
	return nil
}

func (s *statusCommand) checkConnection(status *server.InstanceStatus) (err error) {
	if !s.checkConnected {
		return nil
	}

	if status.ConnectedServer == "" {
		return fmt.Errorf("not connected")
	}

	return nil
}

func (s *statusCommand) checkLastMessage(status *server.InstanceStatus) (err error) {
	if s.lastMessage == 0 {
		return nil
	}

	previous := time.Unix(status.LastMessage, 0)

	if previous.Before(time.Now().Add(-1 * s.lastMessage)) {
		return fmt.Errorf("last message at %v", previous)
	}

	return nil
}

func (s *statusCommand) checkFileAge() (err error) {
	if s.maxAge == 0 {
		return nil
	}

	stat, err := os.Stat(s.statusFile)
	if err != nil {
		return err
	}

	if stat.ModTime().Before(time.Now().Add(-1 * s.maxAge)) {
		return fmt.Errorf("older than %v", s.maxAge)
	}

	return nil
}

func (s *statusCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	rawstatus, err := ioutil.ReadFile(s.statusFile)
	if err != nil {
		s.exit(fmt.Errorf("%s could not be read: %s", s.statusFile, err))
	}

	status := &server.InstanceStatus{}
	err = json.Unmarshal(rawstatus, status)
	if err != nil {
		s.exit(fmt.Errorf("%s could not be parsed: %s", s.statusFile, err))
	}

	err = s.checkFileAge()
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

func (s *statusCommand) exit(err error) {
	if err != nil {
		fmt.Printf("%s %s\n", s.statusFile, err)
		os.Exit(1)
	}

	fmt.Printf("%s OK\n", s.statusFile)
	os.Exit(0)
}

func init() {
	cli.commands = append(cli.commands, &statusCommand{})
}
