// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/tokens"
)

type jWTCreateServerCommand struct {
	file        string
	signingKey  string
	identity    string
	collectives []string
	org         string
	subjects    []string
	submission  bool
	validity    time.Duration
	streamUser  bool
	service     bool
	pk          string

	command
}

func (s *jWTCreateServerCommand) Setup() (err error) {
	if jwt, ok := cmdWithFullCommand("jwt"); ok {
		parts := strings.Split(build.DefaultCollectives, ",")
		collective := "mcollective"
		if len(parts) > 0 {
			collective = parts[0]
		}

		s.cmd = jwt.Cmd().Command("server", "Create a Server JWT token").Alias("s")
		s.cmd.Arg("file", "The JWT file to act on").Required().StringVar(&s.file)
		s.cmd.Arg("identity", "The identity for this server").Required().StringVar(&s.identity)
		s.cmd.Arg("public-key", "Ed25519 public key to embed in the token").Required().StringVar(&s.pk)
		s.cmd.Arg("signing-key", "Path to a private key used to sign the JWT").Required().ExistingFileVar(&s.signingKey)
		s.cmd.Flag("collectives", "Allow the server to access certain collectives").Default(collective).StringsVar(&s.collectives)
		s.cmd.Flag("org", "Adds the user to a specific organization").Default("choria").StringVar(&s.org)
		s.cmd.Flag("subjects", "Additional subjects this node may publish to").StringsVar(&s.subjects)
		s.cmd.Flag("submission", "Enable the node to publish to Choria Streams using Choria Submission").BoolVar(&s.submission)
		s.cmd.Flag("stream-user", "Allow the node to access Choria Streams").BoolVar(&s.streamUser)
		s.cmd.Flag("validity", "How long the token should be valid for").Default("8760h").DurationVar(&s.validity)
		s.cmd.Flag("service", "Indicates that the user can have long validity tokens").BoolVar(&s.service)
	}

	return nil
}

func (s *jWTCreateServerCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (s *jWTCreateServerCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	err = s.createJWT()
	if err != nil {
		return fmt.Errorf("could not create token: %s", err)
	}

	return nil
}

func (s *jWTCreateServerCommand) createJWT() error {
	var err error

	perms := &tokens.ServerPermissions{
		Submission:  s.submission,
		Streams:     s.streamUser,
		ServiceHost: s.service,
	}

	pk, err := hex.DecodeString(s.pk)
	if err != nil {
		return err
	}

	claims, err := tokens.NewServerClaims(s.identity, s.collectives, s.org, perms, s.subjects, pk, "Choria CLI", s.validity)
	if err != nil {
		return err
	}

	err = tokens.SaveAndSignTokenWithKeyFile(claims, s.signingKey, s.file, 0600)
	if err != nil {
		return err
	}

	fmt.Printf("Saved token to %v, use 'choria jwt view %v' to view it\n", s.file, s.file)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &jWTCreateServerCommand{})
}
