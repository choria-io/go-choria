// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/config"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/tokens"
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
	serverProvisioner     bool
	pk                    string
	chain                 bool
	additionalPub         []string
	additionalSub         []string
	useVault              bool

	command
}

func (cl *jWTCreateClientCommand) Setup() (err error) {
	if jwt, ok := cmdWithFullCommand("jwt"); ok {
		cl.cmd = jwt.Cmd().Command("client", "Create a Client JWT token").Alias("c")
		cl.cmd.Arg("file", "The JWT file to act on").Required().StringVar(&cl.file)
		cl.cmd.Arg("identity", "The Caller ID for this user").Required().StringVar(&cl.identity)
		cl.cmd.Arg("signing-key", "Path to a private key used to sign the JWT").Required().StringVar(&cl.signingKey)
		cl.cmd.Flag("agents", "Allow the user to access certain agents").StringsVar(&cl.agents)
		cl.cmd.Flag("org", "Adds the user to a specific organization").Default("choria").StringVar(&cl.org)
		cl.cmd.Flag("opa-file", "Path to a file holding a Open Policy Agent Policy for this user").ExistingFileVar(&cl.opaPolicyFile)
		cl.cmd.Flag("opa", "Open Policy Agent Policy as a string").StringVar(&cl.opaPolicy)
		cl.cmd.Flag("validity", "How long the token should be valid for").Default("1h").DurationVar(&cl.validity)
		cl.cmd.Flag("public-key", "Ed25519 public key to embed in the token").StringVar(&cl.pk)
		cl.cmd.Flag("stream-admin", "Allow the user to administer and use Choria Streams").UnNegatableBoolVar(&cl.streamAdmin)
		cl.cmd.Flag("stream-user", "Allow the user to use Choria Streams").UnNegatableBoolVar(&cl.streamUser)
		cl.cmd.Flag("event-viewer", "Allow the user to view various Choria Events").UnNegatableBoolVar(&cl.eventViewer)
		cl.cmd.Flag("elections-user", "Allow the user to use Choria Elections").UnNegatableBoolVar(&cl.electionUser)
		cl.cmd.Flag("service", "Indicates that the user can have long validity tokens").UnNegatableBoolVar(&cl.service)
		cl.cmd.Flag("system", "Allow the user to access the broker system account").UnNegatableBoolVar(&cl.system)
		cl.cmd.Flag("auth-delegation", "Allow the user to sign requests for other users").UnNegatableBoolVar(&cl.authDelegate)
		cl.cmd.Flag("fleet-management", "Allows access to the Choria fleet using RPC").Default("true").BoolVar(&cl.fleetManagement)
		cl.cmd.Flag("signed-fleet-management", "Requires that all fleet management requests are signed by an authority like AAA Service").UnNegatableBoolVar(&cl.signedFleetManagement)
		cl.cmd.Flag("org-admin", "Allow the user to access all broker traffic").UnNegatableBoolVar(&cl.orgAdmin)
		cl.cmd.Flag("publish", "Additional subjects the user can publish to").StringsVar(&cl.additionalPub)
		cl.cmd.Flag("subscribe", "Additional subjects the user can subscribe to").StringsVar(&cl.additionalSub)
		cl.cmd.Flag("issuer", "Allow this user to sign other users in a chain of trust").UnNegatableBoolVar(&cl.chain)
		cl.cmd.Flag("server-provisioner", "Allows the client to provision servers").UnNegatableBoolVar(&cl.serverProvisioner)
		cl.cmd.Flag("vault", "Use Hashicorp Vault to sign the JWT").UnNegatableBoolVar(&cl.useVault)
	}

	return nil
}

func (cl *jWTCreateClientCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (cl *jWTCreateClientCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	err = cl.createJWT()
	if err != nil {
		return fmt.Errorf("could not create token: %s", err)
	}

	return nil
}

func (cl *jWTCreateClientCommand) createJWT() error {
	var err error

	var opa []byte
	if cl.opaPolicyFile != "" {
		opa, err = os.ReadFile(cl.opaPolicyFile)
		if err != nil {
			return err
		}
	} else if cl.opaPolicy != "" {
		opa = []byte(cl.opaPolicy)
	}

	perms := &tokens.ClientPermissions{
		StreamsAdmin:            cl.streamAdmin,
		StreamsUser:             cl.streamUser,
		EventsViewer:            cl.eventViewer,
		ElectionUser:            cl.electionUser,
		OrgAdmin:                cl.orgAdmin,
		ExtendedServiceLifetime: cl.service,
		SystemUser:              cl.system,
		AuthenticationDelegator: cl.authDelegate,
		FleetManagement:         cl.fleetManagement,
		SignedFleetManagement:   cl.signedFleetManagement,
		ServerProvisioner:       cl.serverProvisioner,
	}

	if cl.chain {
		perms.FleetManagement = false
		perms.SignedFleetManagement = false
		perms.AuthenticationDelegator = false
	}
	if perms.SignedFleetManagement {
		perms.FleetManagement = true
	}
	if perms.ServerProvisioner {
		perms.FleetManagement = true
	}

	pk, err := hex.DecodeString(cl.pk)
	if err != nil {
		return err
	}

	claims, err := tokens.NewClientIDClaims(cl.identity, cl.agents, cl.org, nil, string(opa), "", cl.validity, perms, pk)
	if err != nil {
		return err
	}

	claims.AdditionalSubscribeSubjects = cl.additionalSub
	claims.AdditionalPublishSubjects = cl.additionalPub

	if cl.chain {
		_, sprik, err := iu.Ed25519KeyPairFromSeedFile(cl.signingKey)
		if err != nil {
			return err
		}

		err = claims.AddOrgIssuerData(sprik)
		if err != nil {
			return err
		}
	}

	if cl.useVault {
		var tlsc *tls.Config
		tlsc, err = c.ClientTLSConfig()
		if err == nil {
			to, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			err = tokens.SaveAndSignTokenWithVault(to, claims, cl.signingKey, cl.file, 0600, tlsc, c.Logger("jwt"))
		}
	} else {
		err = tokens.SaveAndSignTokenWithKeyFile(claims, cl.signingKey, cl.file, 0600)
	}
	if err != nil {
		return err
	}

	err = tokens.SaveAndSignTokenWithKeyFile(claims, cl.signingKey, cl.file, 0600)
	if err != nil {
		return err
	}

	fmt.Printf("Saved token to %v, use 'choria jwt view %v' to view it\n", cl.file, cl.file)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &jWTCreateClientCommand{})
}
