// Copyright (c) 2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"github.com/choria-io/go-choria/config"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
)

type tSha256Command struct {
	command

	directory string
	sumsFile  string
}

func init() {
	cli.commands = append(cli.commands, &tSha256Command{})
}

func (r *tSha256Command) Setup() error {
	if machine, ok := cmdWithFullCommand("tool"); ok {
		r.cmd = machine.Cmd().Command("sha256", "Checksums a directory of files recursively using SHA256")
		r.cmd.Arg("dir", "The directory to recursively checksum").Required().ExistingDirVar(&r.directory)
		r.cmd.Flag("validate", "Checksum file used to validate the directory").ExistingFileVar(&r.sumsFile)
	}

	return nil
}

func (r *tSha256Command) Configure() error {
	if debug {
		logrus.SetOutput(os.Stdout)
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Logging at debug level due to CLI override")
	}

	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return err
	}

	cfg.Choria.SecurityProvider = "file"
	cfg.DisableSecurityProviderVerify = true

	return err
}

func (r *tSha256Command) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	if r.sumsFile == "" {
		return r.create()
	}

	return r.validate()
}

func (r *tSha256Command) validate() error {
	ok, err := iu.Sha256VerifyDir(r.sumsFile, r.directory, c.Logger("sha256"), func(file string, ok bool) {
		if !ok {
			fmt.Println(file)
		}
	})
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("directory did not validate using %s", r.sumsFile)
	}

	fmt.Printf("Directory %s validates correctly using %s\n", r.directory, r.sumsFile)

	return nil
}

func (r *tSha256Command) create() error {
	sums, err := iu.Sha256ChecksumDir(r.directory)
	if err != nil {
		return err
	}
	fmt.Print(string(sums))

	return nil
}
