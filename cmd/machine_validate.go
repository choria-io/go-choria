// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/aagent/machine"
)

type mValidateCommand struct {
	command
	sourceDir string
}

func (c *mValidateCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine"); ok {
		c.cmd = machine.Cmd().Command("validate", "Validate the structure and content of a machine.yaml")
		c.cmd.Arg("source", "Directory containing the machine definition").Required().ExistingDirVar(&c.sourceDir)
	}

	return nil
}

func (c *mValidateCommand) Configure() error {
	return nil
}

func (c *mValidateCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	vErrors, err := machine.ValidateDir(c.sourceDir)
	if err != nil {
		return err
	}

	if len(vErrors) == 0 {
		fmt.Printf("%s is has a valid machine.yaml\n", c.sourceDir)
		return nil
	}

	for _, verr := range vErrors {
		fmt.Println(verr)
	}

	return fmt.Errorf("validation failed")
}

func init() {
	cli.commands = append(cli.commands, &mValidateCommand{})
}
