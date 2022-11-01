// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/config"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tokens"
)

type tJWTViewCommand struct {
	file         string
	validateCert string
	json         bool

	command
}

func (v *tJWTViewCommand) Setup() (err error) {
	if jwt, ok := cmdWithFullCommand("jwt"); ok {
		v.cmd = jwt.Cmd().Command("view", "View and Validate Choria JWT tokens").Alias("show").Alias("v").Alias("s").Default()
		v.cmd.Arg("file", "The JWT file to act on").Required().ExistingFileVar(&v.file)
		v.cmd.Arg("certificate", "Path to a certificate used to validate or sign the JWT").ExistingFileVar(&v.validateCert)
		v.cmd.Flag("json", "Render the token as JSON").UnNegatableBoolVar(&v.json)
	}

	return nil
}

func (v *tJWTViewCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (v *tJWTViewCommand) validateServerToken(token string) error {
	var claims *tokens.ServerClaims
	var err error
	var validated bool

	if v.validateCert == "" {
		claims, err = tokens.ParseServerTokenUnverified(token)
	} else {
		claims, err = tokens.ParseServerTokenWithKeyfile(token, v.validateCert)
		validated = true
	}
	if err != nil {
		return fmt.Errorf("could not parse token: %s", err)
	}

	if validated {
		fmt.Printf("Validated Server Token %s\n\n", v.file)
	} else {
		fmt.Printf("Unvalidated Server Token %s\n\n", v.file)
	}

	fmt.Printf("             Identity: %s\n", claims.ChoriaIdentity)
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		fmt.Printf("           Expires At: %s (expired %s ago)\n", claims.ExpiresAt.Time, iu.RenderDuration(time.Since(claims.ExpiresAt.Time)))
	} else if claims.ExpiresAt != nil {
		fmt.Printf("           Expires At: %s (%s)\n", claims.ExpiresAt.Time, iu.RenderDuration(time.Until(claims.ExpiresAt.Time)))
	}
	fmt.Printf("          Collectives: %s\n", strings.Join(claims.Collectives, ", "))
	fmt.Printf("           Public Key: %s\n", claims.PublicKey)
	fmt.Printf("    Organization Unit: %s\n", claims.OrganizationUnit)
	if len(claims.AdditionalPublishSubjects) > 0 {
		fmt.Printf("  Additional Subjects: %s\n", strings.Join(claims.AdditionalPublishSubjects, ", "))
	}

	_, uid := claims.UniqueID()
	fmt.Printf("   Private Network ID: %s\n", uid)

	if claims.Permissions != nil {
		fmt.Println("   Broker Permissions:")
		if claims.Permissions.ServiceHost {
			fmt.Println("          Can host services")
		}
		if claims.Permissions.Submission {
			fmt.Println("          Can publish Choria Submission messages")
		}
		if claims.Permissions.Streams {
			fmt.Println("          Can access Choria Streams")
		}
	}

	stdc, err := json.MarshalIndent(claims.StandardClaims, strings.Repeat(" ", 23), "  ")
	if err != nil {
		return nil
	}
	fmt.Printf("      Standard Claims: %s\n", string(stdc))

	return nil
}

func (v *tJWTViewCommand) validateProvisionToken(token string) error {
	var claims *tokens.ProvisioningClaims
	var err error
	var validated bool

	if v.validateCert == "" {
		claims, err = tokens.ParseProvisionTokenUnverified(token)
	} else {
		claims, err = tokens.ParseProvisioningTokenWithKeyfile(token, v.validateCert)
		validated = true
	}
	if err != nil {
		return fmt.Errorf("could not parse token: %s", err)
	}

	if claims.Token != "" {
		claims.Token = "*****"
	}

	if validated {
		fmt.Printf("Validated Provisioning Token %s\n\n", v.file)
	} else {
		fmt.Printf("Unvalidated Provisioning Token %s\n\n", v.file)
	}

	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		fmt.Printf("                    Expires At: %s (expired %s ago)\n", claims.ExpiresAt.Time, iu.RenderDuration(time.Since(claims.ExpiresAt.Time)))
	} else if claims.ExpiresAt != nil {
		fmt.Printf("                    Expires At: %s (%s)\n", claims.ExpiresAt.Time, iu.RenderDuration(time.Until(claims.ExpiresAt.Time)))
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
	if len(claims.Extensions) > 0 {
		ext, err := json.MarshalIndent(claims.Extensions, "                                ", "  ")
		if err != nil {
			return nil
		}
		fmt.Printf("                    Extensions: %s\n", string(ext))

	}

	stdc, err := json.MarshalIndent(claims.StandardClaims, "                                ", "  ")
	if err != nil {
		return nil
	}
	fmt.Printf("               Standard Claims: %s\n", string(stdc))

	return nil
}

func (v *tJWTViewCommand) validateClientToken(token string) error {
	var claims *tokens.ClientIDClaims
	var err error
	var validated bool

	if v.validateCert == "" {
		claims, err = tokens.ParseClientIDTokenUnverified(token)
	} else {
		claims, err = tokens.ParseClientIDTokenWithKeyfile(token, v.validateCert, true)
		validated = true
	}
	if err != nil {
		return fmt.Errorf("could not parse token: %s", err)
	}

	if validated {
		fmt.Printf("Validated Client Identification Token %s\n\n", v.file)
	} else {
		fmt.Printf("Unvalidated Client Identification Token %s\n\n", v.file)
	}

	fmt.Printf("          Caller ID: %s\n", claims.CallerID)
	if claims.OrganizationUnit != "" {
		fmt.Printf("  Organization Unit: %s\n", claims.OrganizationUnit)
	}
	if len(claims.AllowedAgents) > 0 {
		fmt.Printf("     Allowed Agents: %s\n", strings.Join(claims.AllowedAgents, ", "))
	}
	fmt.Printf("         Public Key: %s\n", claims.PublicKey)
	_, uid := claims.UniqueID()
	fmt.Printf(" Private Network ID: %s\n", uid)
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		fmt.Printf("         Expires At: %s (expired %s ago)\n", claims.ExpiresAt.Time, iu.RenderDuration(time.Since(claims.ExpiresAt.Time)))
	} else if claims.ExpiresAt != nil {
		fmt.Printf("         Expires At: %s (%s)\n", claims.ExpiresAt.Time, iu.RenderDuration(time.Until(claims.ExpiresAt.Time)))
	}
	if len(claims.AdditionalSubscribeSubjects) > 0 {
		fmt.Printf(" Subscribe Subjects: %s\n", strings.Join(claims.AdditionalSubscribeSubjects, ", "))
	}
	if len(claims.AdditionalPublishSubjects) > 0 {
		fmt.Printf("   Publish Subjects: %s\n", strings.Join(claims.AdditionalPublishSubjects, ", "))
	}
	if claims.Permissions != nil {
		fmt.Println()
		fmt.Println(" Client Permissions:")
		if claims.Permissions.FleetManagement || claims.Permissions.SignedFleetManagement {
			if claims.Permissions.SignedFleetManagement {
				fmt.Println("      Can manage Choria fleet nodes subject to authorizing signature")
			} else {
				fmt.Println("      Can manage Choria fleet nodes")
			}
		}
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
		if claims.Permissions.SystemUser {
			fmt.Println("      Can access the Broker system account")
		}
		if claims.Permissions.AuthenticationDelegator {
			fmt.Println("      Can sign requests on behalf of other users")
		}
		fmt.Println()
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

func (v *tJWTViewCommand) validateAnyToken(token string) error {
	claims, err := tokens.ParseTokenUnverified(token)
	if err != nil {
		return err
	}

	return iu.DumpJSONIndent(claims)
}

func (v *tJWTViewCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	token, err := os.ReadFile(v.file)
	if err != nil {
		return fmt.Errorf("could not read token: %s", err)
	}

	ts := string(token)
	purpose := tokens.TokenPurpose(ts)

	switch {
	case !v.json && purpose == tokens.ProvisioningPurpose:
		return v.validateProvisionToken(ts)

	case !v.json && purpose == tokens.ClientIDPurpose:
		return v.validateClientToken(ts)

	case !v.json && purpose == tokens.ServerPurpose:
		return v.validateServerToken(ts)

	default:
		return v.validateAnyToken(ts)
	}
}

func init() {
	cli.commands = append(cli.commands, &tJWTViewCommand{})
}
