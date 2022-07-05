// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"
)

type jWTCommand struct {
	command
}

func (j *jWTCommand) Setup() (err error) {
	j.cmd = cli.app.Command("jwt", "Create, Validate and View Choria JWT files")

	return nil
}

func (j *jWTCommand) Configure() error {
	return nil
}

func (j *jWTCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &jWTCommand{})
}
