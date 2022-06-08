// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/tokens"
)

type jWTCreateClientCommand struct {
	file          string
	signingKey    string
	identity      string
	agents        []string
	org           string
	opaPolicyFile string
	opaPolicy     string
	validity      time.Duration
	streamAdmin   bool
	streamUser    bool
	eventViewer   bool
	electionUser  bool
	orgAdmin      bool
	service       bool
	system        bool
	pk            string

	command
}

func (c *jWTCreateClientCommand) Setup() (err error) {
	if jwt, ok := cmdWithFullCommand("jwt"); ok {
		c.cmd = jwt.Cmd().Command("client", "Create a Client JWT token").Alias("c")
		c.cmd.Arg("file", "The JWT file to act on").Required().StringVar(&c.file)
		c.cmd.Arg("identity", "The Caller ID for this user").Required().StringVar(&c.identity)
		c.cmd.Arg("signing-key", "Path to a private key used to sign the JWT").Required().ExistingFileVar(&c.signingKey)
		c.cmd.Flag("agents", "Allow the user to access certain agents").StringsVar(&c.agents)
		c.cmd.Flag("org", "Adds the user to a specific organization").Default("choria").StringVar(&c.org)
		c.cmd.Flag("opa-file", "Path to a file holding a Open Policy Agent Policy for this user").ExistingFileVar(&c.opaPolicyFile)
		c.cmd.Flag("opa", "Open Policy Agent Policy as a string").StringVar(&c.opaPolicy)
		c.cmd.Flag("validity", "How long the token should be valid for").Default("1h").DurationVar(&c.validity)
		c.cmd.Flag("public-key", "Ed25519 public key to embed in the token").StringVar(&c.pk)
		c.cmd.Flag("stream-admin", "Allow the user to administer and use Choria Streams").BoolVar(&c.streamAdmin)
		c.cmd.Flag("stream-user", "Allow the user to use Choria Streams").BoolVar(&c.streamUser)
		c.cmd.Flag("event-viewer", "Allow the user to view various Choria Events").BoolVar(&c.eventViewer)
		c.cmd.Flag("elections-user", "Allow the user to use Choria Elections").BoolVar(&c.electionUser)
		c.cmd.Flag("service", "Indicates that the user can have long validity tokens").BoolVar(&c.service)
		c.cmd.Flag("system", "Allow the user to access the broker system account").BoolVar(&c.system)
	}

	return nil
}

func (c *jWTCreateClientCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (c *jWTCreateClientCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	err = c.createJWT()
	if err != nil {
		return fmt.Errorf("could not create token: %s", err)
	}

	return nil
}

func (c *jWTCreateClientCommand) createJWT() error {
	var err error

	var opa []byte
	if c.opaPolicyFile != "" {
		opa, err = os.ReadFile(c.opaPolicyFile)
		if err != nil {
			return err
		}
	} else if c.opaPolicy != "" {
		opa = []byte(c.opaPolicy)
	}

	perms := &tokens.ClientPermissions{
		StreamsAdmin:            c.streamAdmin,
		StreamsUser:             c.streamUser,
		EventsViewer:            c.eventViewer,
		ElectionUser:            c.electionUser,
		OrgAdmin:                c.orgAdmin,
		ExtendedServiceLifetime: c.service,
		SystemUser:              c.system,
	}

	claims, err := tokens.NewClientIDClaims(c.identity, c.agents, c.org, nil, string(opa), "Choria CLI", c.validity, perms, []byte(c.pk))
	if err != nil {
		return err
	}

	err = tokens.SaveAndSignTokenWithKeyFile(claims, c.signingKey, c.file, 0600)
	if err != nil {
		return err
	}

	fmt.Printf("Saved token to %v, use 'choria jwt view %v' to view it\n", c.file, c.file)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &jWTCreateClientCommand{})
}
