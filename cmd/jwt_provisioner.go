// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

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
	urls        []string

	command
}

func (p *jWTCreateProvCommand) Setup() (err error) {
	if jwt, ok := cmdWithFullCommand("jwt"); ok {
		p.cmd = jwt.Cmd().Command("provisioning", "Create a Provisioning JWT token").Alias("prov").Alias("provision").Alias("p")
		p.cmd.Arg("file", "The JWT file to act on").Required().StringVar(&p.file)
		p.cmd.Arg("signing-key", "Path to a private key used to sign the JWT").Required().ExistingFileVar(&p.signingKey)
		p.cmd.Flag("insecure", "Disable TLS security during provisioning").BoolVar(&p.insecure)
		p.cmd.Flag("token", "Token used to secure access to the provisioning agent").StringVar(&p.token)
		p.cmd.Flag("urls", "URLs to connect to for provisioning").StringsVar(&p.urls)
		p.cmd.Flag("srv", "Domain to query for SRV records to find provisioning urls").StringVar(&p.srvDomain)
		p.cmd.Flag("default", "Enables provisioning by default").BoolVar(&p.provDefault)
		p.cmd.Flag("registration", "File to publish as registration data during provisioning").StringVar(&p.regData)
		p.cmd.Flag("facts", "File to use for facts during registration").StringVar(&p.facts)
		p.cmd.Flag("username", "Username to connect to the provisioning broker with").StringVar(&p.uname)
		p.cmd.Flag("password", "Password to connect to the provisioning broker with").StringVar(&p.password)
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
	claims, err := tokens.NewProvisioningClaims(!p.insecure, p.provDefault, p.token, p.uname, p.password, p.urls, p.srvDomain, p.regData, p.facts, "Choria CLI", 0)
	if err != nil {
		return err
	}

	err = tokens.SaveAndSignTokenWithKeyFile(claims, p.signingKey, p.file, 0600)
	if err != nil {
		return err
	}

	fmt.Printf("Saved token to %v, use 'choria jwt view %v' to view it\n", p.file, p.file)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &jWTCreateProvCommand{})
}
