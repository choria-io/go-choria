// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tokens"
)

type tJWTCommand struct {
	file         string
	insecure     bool
	validateCert string
	token        string
	srvDomain    string
	regData      string
	facts        string
	uname        string
	password     string
	provDefault  bool
	urls         []string

	command
}

func (j *tJWTCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		j.cmd = tool.Cmd().Command("jwt", "Create or validate JWT files")
		j.cmd.Arg("file", "The JWT file to act on").Required().StringVar(&j.file)
		j.cmd.Arg("certificate", "Path to a certificate used to validate or sign the JWT").ExistingFileVar(&j.validateCert)
		j.cmd.Flag("insecure", "Disable TLS security during provisioning").BoolVar(&j.insecure)
		j.cmd.Flag("token", "Token used to secure access to the provisioning agent").StringVar(&j.token)
		j.cmd.Flag("urls", "URLs to connect to for provisioning").StringsVar(&j.urls)
		j.cmd.Flag("srv", "Domain to query for SRV records to find provisioning urls").StringVar(&j.srvDomain)
		j.cmd.Flag("default", "Enables provisioning by default").BoolVar(&j.provDefault)
		j.cmd.Flag("registration", "File to publish as registration data during provisioning").StringVar(&j.regData)
		j.cmd.Flag("facts", "File to use for facts during registration").StringVar(&j.facts)
		j.cmd.Flag("username", "Username to connect to the provisioning broker with").StringVar(&j.uname)
		j.cmd.Flag("password", "Password to connect to the provisioning broker with").StringVar(&j.password)
	}

	return nil
}

func (j *tJWTCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (j *tJWTCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if choria.FileExist(j.file) {
		err = j.validateJWT()
		if err != nil {
			return fmt.Errorf("token validation failed: %s", err)
		}
	} else {
		err = j.createJWT()
		if err != nil {
			return fmt.Errorf("could not create token: %s", err)
		}
	}

	return nil
}

func (j *tJWTCommand) validateProvisionToken(token string) error {
	var claims *tokens.ProvisioningClaims
	var err error
	var validated bool

	if j.validateCert == "" {
		claims, err = tokens.ParseProvisionTokenUnverified(token)
	} else {
		claims, err = tokens.ParseProvisioningTokenWithKeyfile(token, j.validateCert)
		validated = true
	}
	if err != nil {
		return fmt.Errorf("could not parse token: %s", err)
	}

	if claims.Token != "" {
		claims.Token = "*****"
	}

	if validated {
		fmt.Printf("Validated Provisioning Token %s\n\n", j.file)
	} else {
		fmt.Printf("Provisioning Token %s\n\n", j.file)
	}

	fmt.Printf("                         Token: %s\n", claims.Token)
	fmt.Printf("                        Secure: %t\n", claims.Secure)
	if claims.SRVDomain != "" {
		fmt.Printf("                    SRV Domain: %s\n", claims.SRVDomain)
	}
	if claims.URLs != "" {
		fmt.Printf("                          URLS: %s\n", claims.URLs)
	}
	fmt.Printf("       Provisioning by default: %t\n", claims.ProvDefault)
	if claims.ProvRegData != "" {
		fmt.Printf("Provisioning Registration Data: %s\n", claims.ProvRegData)
	}
	if claims.ProvFacts != "" {
		fmt.Printf("            Provisioning Facts: %s\n", claims.ProvFacts)
	}
	if claims.ProvNatsUser != "" {
		fmt.Printf("               Broker Username: %s\n", claims.ProvNatsUser)
	}
	if claims.ProvNatsPass != "" {
		fmt.Println("               Broker Password: *****")
	}

	stdc, err := json.MarshalIndent(claims.StandardClaims, "                               ", "  ")
	if err != nil {
		return nil
	}

	fmt.Printf("               Standard Claims: %s\n", string(stdc))

	return nil
}

func (j *tJWTCommand) validateClientToken(token string) error {
	var claims *tokens.ClientIDClaims
	var err error
	var validated bool

	if j.validateCert == "" {
		claims, err = tokens.ParseClientIDTokenUnverified(token)
	} else {
		claims, err = tokens.ParseClientIDTokenWithKeyfile(token, j.validateCert, true)
		validated = true
	}
	if err != nil {
		return fmt.Errorf("could not parse token: %s", err)
	}

	if validated {
		fmt.Printf("Validated Client Identification Token %s\n\n", j.file)
	} else {
		fmt.Printf("Client Identification Token %s\n\n", j.file)
	}

	fmt.Printf("          Caller ID: %s\n", claims.CallerID)
	if claims.OrganizationUnit != "" {
		fmt.Printf("  Organization Unit: %s\n", claims.OrganizationUnit)
	}
	if len(claims.AllowedAgents) > 0 {
		fmt.Printf("     Allowed Agents: %s\n", strings.Join(claims.AllowedAgents, ", "))
	}
	if claims.Permissions != nil {
		fmt.Println(" Broker Permissions:")
		if claims.Permissions.ElectionUser {
			fmt.Println("      Can use Leader Elections")
		}
		if claims.Permissions.EventsViewer {
			fmt.Println("      Can view Lifecycle and Autonomous Agent events")
		}
		if claims.Permissions.StreamsUser {
			fmt.Println("      Can use Choria Streams")
		}
		if claims.Permissions.StreamsAdmin {
			fmt.Println("      Can administer Choria Streams")
		}
		if claims.Permissions.OrgAdmin {
			fmt.Println("      Can observe all traffic on all subjects")
		}
	}

	if len(claims.UserProperties) > 0 {
		jc, err := json.MarshalIndent(claims.UserProperties, strings.Repeat(" ", 21), "  ")
		if err != nil {
			return nil
		}
		fmt.Printf("    User Properties: %s\n", string(jc))
	}

	jc, err := json.MarshalIndent(claims.StandardClaims, strings.Repeat(" ", 21), "  ")
	if err != nil {
		return nil
	}
	fmt.Printf("    Standard Claims: %s\n", string(jc))

	if len(claims.OPAPolicy) > 0 {
		padding := strings.Repeat(" ", 21)
		lines := strings.Split(claims.OPAPolicy, "\n")
		fmt.Printf("         OPA Policy: %s\n", lines[0])
		for _, line := range lines[1:] {
			fmt.Printf("%s%s\n", padding, line)
		}
	}

	return nil
}

func (j *tJWTCommand) validateAnyToken(token string) error {
	claims, err := tokens.ParseTokenUnverified(token)
	if err != nil {
		return err
	}

	return iu.DumpJSONIndent(claims)
}

func (j *tJWTCommand) validateJWT() error {
	token, err := os.ReadFile(j.file)
	if err != nil {
		return fmt.Errorf("could not read token: %s", err)
	}

	ts := string(token)

	switch tokens.TokenPurpose(ts) {
	case tokens.ProvisioningPurpose:
		return j.validateProvisionToken(ts)

	case tokens.ClientIDPurpose:
		return j.validateClientToken(ts)

	default:
		return j.validateAnyToken(ts)
	}
}

func (j *tJWTCommand) createJWT() error {
	if j.token == "" {
		survey.AskOne(&survey.Password{Message: "Provisioning Token"}, &j.token, survey.WithValidator(survey.Required))
	}

	if j.srvDomain == "" && len(j.urls) == 0 {
		return fmt.Errorf("URLs or a SRV Domain is required")
	}
	claims, err := tokens.NewProvisioningClaims(!j.insecure, j.provDefault, j.token, j.uname, j.password, j.urls, j.srvDomain, j.regData, j.facts, "choria cli", 0)
	if err != nil {
		return err
	}

	err = tokens.SaveAndSignTokenWithKeyFile(claims, j.validateCert, j.file, 0600)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tJWTCommand{})
}
