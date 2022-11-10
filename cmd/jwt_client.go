// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/config"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tokens"
)

type jWTCreateClientCommand struct {
	file                  string
	signingKey            string
	identity              string
	agents                []string
	org                   string
	opaPolicyFile         string
	opaPolicy             string
	validity              time.Duration
	streamAdmin           bool
	streamUser            bool
	eventViewer           bool
	electionUser          bool
	orgAdmin              bool
	service               bool
	system                bool
	authDelegate          bool
	fleetManagement       bool
	signedFleetManagement bool
	pk                    string
	chain                 bool
	additionalPub         []string
	additionalSub         []string

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
		c.cmd.Flag("stream-admin", "Allow the user to administer and use Choria Streams").UnNegatableBoolVar(&c.streamAdmin)
		c.cmd.Flag("stream-user", "Allow the user to use Choria Streams").UnNegatableBoolVar(&c.streamUser)
		c.cmd.Flag("event-viewer", "Allow the user to view various Choria Events").UnNegatableBoolVar(&c.eventViewer)
		c.cmd.Flag("elections-user", "Allow the user to use Choria Elections").UnNegatableBoolVar(&c.electionUser)
		c.cmd.Flag("service", "Indicates that the user can have long validity tokens").UnNegatableBoolVar(&c.service)
		c.cmd.Flag("system", "Allow the user to access the broker system account").UnNegatableBoolVar(&c.system)
		c.cmd.Flag("auth-delegation", "Allow the user to sign requests for other users").UnNegatableBoolVar(&c.authDelegate)
		c.cmd.Flag("fleet-management", "Allows access to the Choria fleet using RPC").Default("true").BoolVar(&c.fleetManagement)
		c.cmd.Flag("signed-fleet-management", "Requires that all fleet management requests are signed by an authority like AAA Service").UnNegatableBoolVar(&c.signedFleetManagement)
		c.cmd.Flag("org-admin", "Allow the user to access all broker traffic").UnNegatableBoolVar(&c.orgAdmin)
		c.cmd.Flag("publish", "Additional subjects the user can publish to").StringsVar(&c.additionalPub)
		c.cmd.Flag("subscribe", "Additional subjects the user can subscribe to").StringsVar(&c.additionalSub)
		c.cmd.Flag("issuer", "Allow this user to sign other users in a chain of trust").UnNegatableBoolVar(&c.chain)
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
		AuthenticationDelegator: c.authDelegate,
		FleetManagement:         c.fleetManagement,
		SignedFleetManagement:   c.signedFleetManagement,
	}

	pk, err := hex.DecodeString(c.pk)
	if err != nil {
		return err
	}

	claims, err := tokens.NewClientIDClaims(c.identity, c.agents, c.org, nil, string(opa), "", c.validity, perms, pk)
	if err != nil {
		return err
	}

	claims.AdditionalSubscribeSubjects = c.additionalSub
	claims.AdditionalPublishSubjects = c.additionalPub

	if c.chain {
		spubk, sprik, err := iu.Ed25519KeyPairFromSeedFile(c.signingKey)
		if err != nil {
			return err
		}

		dat, err := claims.OrgIssuerChainData()
		if err != nil {
			return fmt.Errorf("could not determine chain data to sign: %w", err)
		}

		sig, err := iu.Ed25519Sign(sprik, dat)
		if err != nil {
			return err
		}

		claims.SetOrgIssuer(spubk)
		claims.SetChainIssuerTrustSignature(sig)
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
