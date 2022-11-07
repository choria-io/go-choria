// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/submission"
)

type tSubmitCommand struct {
	command
	subject     string
	payloadFile string
	reliable    bool
	priority    string
	ttl         time.Duration
	maxTries    uint
	sender      string
	sign        bool
}

func (s *tSubmitCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		s.cmd = tool.Cmd().Command("submit", "Submit a message to the Submission system")
		s.cmd.Arg("subject", "The subject to publish to").Required().StringVar(&s.subject)
		s.cmd.Arg("payload", "The file to read as payload, - for STDIN").Required().StringVar(&s.payloadFile)
		s.cmd.Flag("reliable", "Marks the message as reliable").UnNegatableBoolVar(&s.reliable)
		s.cmd.Flag("priority", "The message priority").Default("4").EnumVar(&s.priority, "0", "1", "2", "3", "4")
		s.cmd.Flag("ttl", "The maximum time this message is valid for as duration").Default("24h").DurationVar(&s.ttl)
		s.cmd.Flag("tries", "Maximum amount of attempts made to deliver this message").Default("100").UintVar(&s.maxTries)
		s.cmd.Flag("sender", "The sender of the message").Default(fmt.Sprintf("user %d", os.Geteuid())).StringVar(&s.sender)
		s.cmd.Flag("sign", "Request that the server signs the message when publishing").UnNegatableBoolVar(&s.sign)
	}

	return nil
}

func (s *tSubmitCommand) Configure() (err error) {
	err = commonConfigure()
	if err != nil {
		cfg, err = config.NewDefaultConfig()
		if err != nil {
			return err
		}
		cfg.Choria.SecurityProvider = "file"
	}

	cfg.DisableSecurityProviderVerify = true

	return err
}

func (s *tSubmitCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	subm, err := submission.NewFromChoria(c, submission.Directory)
	if err != nil {
		return err
	}

	prio, _ := strconv.Atoi(s.priority)
	msg := subm.NewMessage()
	msg.Subject = s.subject
	msg.Reliable = s.reliable
	msg.Priority = uint(prio)
	msg.TTL = s.ttl.Seconds()
	msg.MaxTries = s.maxTries
	msg.Sender = s.sender
	msg.Sign = s.sign

	if s.payloadFile == "-" {
		msg.Payload, err = io.ReadAll(os.Stdin)
	} else {
		if !util.FileExist(s.payloadFile) {
			return fmt.Errorf("payload %s does not exist", s.payloadFile)
		}

		msg.Payload, err = os.ReadFile(s.payloadFile)
		if err != nil {
			return err
		}
	}

	if len(msg.Payload) == 0 {
		return fmt.Errorf("payload is empty")
	}

	err = subm.Submit(msg)
	if err != nil {
		return err
	}

	fmt.Println(msg.ID)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tSubmitCommand{})
}
