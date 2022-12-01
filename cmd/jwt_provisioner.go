// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/tokens"
)

type jWTCreateProvCommand struct {
	file        string
	insecure    bool
	signingKey  string
	token       string
	srvDomain   string
	regData     string
	facts       string
	uname       string
	password    string
	provDefault bool
	extensions  string
	urls        []string
	org         string
	useVault    bool
	v2          bool
	allowUpdate bool

	command
}

func (p *jWTCreateProvCommand) Setup() (err error) {
	if jwt, ok := cmdWithFullCommand("jwt"); ok {
		p.cmd = jwt.Cmd().Command("provisioning", "Create a Provisioning JWT token").Alias("prov").Alias("provision").Alias("p")
		p.cmd.Arg("file", "The JWT file to act on").Required().StringVar(&p.file)
		p.cmd.Arg("signing-key", "Path to a private key used to sign the JWT").Required().StringVar(&p.signingKey)
		p.cmd.Flag("insecure", "Disable TLS security during provisioning").Default("true").BoolVar(&p.insecure)
		p.cmd.Flag("token", "Token used to secure access to the provisioning agent").StringVar(&p.token)
		p.cmd.Flag("urls", "URLs to connect to for provisioning").StringsVar(&p.urls)
		p.cmd.Flag("srv", "Domain to query for SRV records to find provisioning urls").StringVar(&p.srvDomain)
		p.cmd.Flag("default", "Enables provisioning by default").UnNegatableBoolVar(&p.provDefault)
		p.cmd.Flag("registration", "File to publish as registration data during provisioning").StringVar(&p.regData)
		p.cmd.Flag("facts", "File to use for facts during registration").StringVar(&p.facts)
		p.cmd.Flag("username", "Username to connect to the provisioning broker with").StringVar(&p.uname)
		p.cmd.Flag("password", "Password to connect to the provisioning broker with").StringVar(&p.password)
		p.cmd.Flag("extensions", "Adds additional extensions to the token, accepts JSON data").PlaceHolder("JSON").StringVar(&p.extensions)
		p.cmd.Flag("org", "Adds the node to a specific organization for trust validation").Default("choria").StringVar(&p.org)
		p.cmd.Flag("vault", "Use Hashicorp Vault to sign the JWT").UnNegatableBoolVar(&p.useVault)
		p.cmd.Flag("protocol-v2", "Use version 2 network protocol and security").UnNegatableBoolVar(&p.v2)
		p.cmd.Flag("update", "Allow over the air server updates from the Choria Provisioner").UnNegatableBoolVar(&p.allowUpdate)
	}

	return nil
}

func (p *jWTCreateProvCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (p *jWTCreateProvCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	err = p.createJWT()
	if err != nil {
		return fmt.Errorf("could not create token: %s", err)
	}

	return nil
}

func (p *jWTCreateProvCommand) createJWT() error {
	if p.token == "" {
		survey.AskOne(&survey.Password{Message: "Provisioning Access Token"}, &p.token, survey.WithValidator(survey.Required))
	}

	if p.srvDomain == "" && len(p.urls) == 0 {
		return fmt.Errorf("URLs or a SRV Domain is required")
	}
	claims, err := tokens.NewProvisioningClaims(!p.insecure, p.provDefault, p.token, p.uname, p.password, p.urls, p.srvDomain, p.regData, p.facts, p.org, "", 0)
	if err != nil {
		return err
	}

	if p.extensions != "" {
		ext := tokens.MapClaims{}
		err := json.Unmarshal([]byte(p.extensions), &ext)
		if err != nil {
			return fmt.Errorf("invalid extensions: %v", err)
		}
		claims.Extensions = ext
	}

	claims.ProtoV2 = p.v2
	claims.AllowUpdate = p.allowUpdate

	if p.useVault {
		var tlsc *tls.Config
		tlsc, err = c.ClientTLSConfig()
		if err == nil {
			to, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			err = tokens.SaveAndSignTokenWithVault(to, claims, p.signingKey, p.file, 0600, tlsc, c.Logger("jwt"))
		}
	} else {
		err = tokens.SaveAndSignTokenWithKeyFile(claims, p.signingKey, p.file, 0600)
	}
	if err != nil {
		return err
	}

	fmt.Printf("Saved token to %v, use 'choria jwt view %v' to view it\n", p.file, p.file)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &jWTCreateProvCommand{})
}
