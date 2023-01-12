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
	"github.com/choria-io/tokens"
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
		v.cmd.Arg("public", "Path to a public key used to validate or sign the JWT").ExistingFileVar(&v.validateCert)
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
	exp := claims.ExpireTime()
	if !exp.IsZero() && time.Now().After(exp) {
		fmt.Printf("           Expires At: %s (expired %s ago)\n", exp, iu.RenderDuration(time.Since(exp)))
	} else if !exp.IsZero() {
		fmt.Printf("           Expires At: %s (%s)\n", exp, iu.RenderDuration(time.Until(exp)))
	}
	fmt.Printf("          Collectives: %s\n", strings.Join(claims.Collectives, ", "))
	fmt.Printf("           Public Key: %s\n", claims.PublicKey)
	fmt.Printf("    Organization Unit: %s\n", claims.OrganizationUnit)
	if len(claims.AdditionalPublishSubjects) > 0 {
		fmt.Printf("  Additional Subjects: %s\n", strings.Join(claims.AdditionalPublishSubjects, ", "))
	}

	_, uid := claims.UniqueID()
	fmt.Printf("   Private Network ID: %s\n", uid)

	tcm, err := v.trustChainDescription(claims.StandardClaims)
	if err != nil {
		return err
	}
	if tcm != "" {
		fmt.Printf("          Trust Chain: %s\n", tcm)
	}

	if claims.Permissions != nil {
		perms := []string{}

		if claims.Permissions.ServiceHost {
			perms = append(perms, "          Can host services")
		}
		if claims.Permissions.Submission {
			perms = append(perms, "          Can publish Choria Submission messages")
		}
		if claims.Permissions.Streams {
			perms = append(perms, "          Can access Choria Streams")
		}
		if claims.Permissions.Governor {
			perms = append(perms, "          Can access Choria Governor")
		}

		if len(perms) == 0 {
			perms = append(perms, "          No server specific permissions granted")
		}

		fmt.Println()
		fmt.Println("   Broker Permissions:")
		fmt.Println()

		for _, p := range perms {
			fmt.Println(p)
		}

		fmt.Println()
	}

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

	exp := claims.ExpireTime()
	if !exp.IsZero() && time.Now().After(exp) {
		fmt.Printf("                    Expires At: %s (expired %s ago)\n", exp, iu.RenderDuration(time.Since(exp)))
	} else if !exp.IsZero() {
		fmt.Printf("                    Expires At: %s (%s)\n", exp, iu.RenderDuration(time.Until(exp)))
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
	if claims.ProtoV2 {
		fmt.Println("      Using version 2 Protocol: true")
	}
	fmt.Printf("       Server Version Upgrades: %t\n", claims.AllowUpdate)

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

func (v *tJWTViewCommand) trustChainDescription(claims tokens.StandardClaims) (string, error) {
	if claims.Issuer == "" || claims.TrustChainSignature == "" {
		return "", nil
	}

	if claims.IsChainedIssuer(false) {
		if claims.IsChainedIssuer(true) {
			return fmt.Sprintf("Can Issue Clients as part of a trust chain with Issuer %s", strings.TrimPrefix(claims.Issuer, tokens.OrgIssuerPrefix)), nil
		}

		return "Invalid signing data, issued users will be invalid", nil
	}

	id, _, _, _, err := claims.ParseChainIssuerData()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Issued by %s", id), nil
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

	exp := claims.ExpireTime()
	if !exp.IsZero() && time.Now().After(exp) {
		fmt.Printf("         Expires At: %s (expired %s ago)\n", exp, iu.RenderDuration(time.Since(exp)))
	} else if !exp.IsZero() {
		fmt.Printf("         Expires At: %s (%s)\n", exp, iu.RenderDuration(time.Until(exp)))
	}
	if len(claims.AdditionalSubscribeSubjects) > 0 {
		fmt.Printf(" Subscribe Subjects: %s\n", strings.Join(claims.AdditionalSubscribeSubjects, ", "))
	}
	if len(claims.AdditionalPublishSubjects) > 0 {
		fmt.Printf("   Publish Subjects: %s\n", strings.Join(claims.AdditionalPublishSubjects, ", "))
	}

	tcm, err := v.trustChainDescription(claims.StandardClaims)
	if err != nil {
		return err
	}
	if tcm != "" {
		fmt.Printf("        Trust Chain: %s\n", tcm)
	}

	if claims.Permissions != nil {
		perms := []string{}
		if claims.Permissions.FleetManagement || claims.Permissions.SignedFleetManagement {
			if claims.Permissions.SignedFleetManagement {
				perms = append(perms, "      Can manage Choria fleet nodes subject to authorizing signature")
			} else {
				perms = append(perms, "      Can manage Choria fleet nodes")
			}
		}
		if claims.Permissions.ElectionUser {
			perms = append(perms, "      Can use Leader Elections")
		}
		if claims.Permissions.EventsViewer {
			perms = append(perms, "      Can view Lifecycle and Autonomous Agent events")
		}
		if claims.Permissions.StreamsUser {
			perms = append(perms, "      Can use Choria Streams")
		}
		if claims.Permissions.StreamsAdmin {
			perms = append(perms, "      Can administer Choria Streams")
		}
		if claims.Permissions.Governor {
			perms = append(perms, "      Can access Choria Governors")
		}
		if claims.Permissions.OrgAdmin {
			perms = append(perms, "      Can observe all traffic on all subjects and access the system account")
		}
		if claims.Permissions.SystemUser {
			perms = append(perms, "      Can access the Broker system account")
		}
		if claims.Permissions.AuthenticationDelegator {
			perms = append(perms, "      Can sign requests on behalf of other users")
		}
		if claims.Permissions.ExtendedServiceLifetime {
			perms = append(perms, "      Can have an extended token lifetime")
		}
		if claims.Permissions.ServerProvisioner {
			perms = append(perms, "      Can provision Choria Servers")
		}

		fmt.Println()
		fmt.Println(" Client Permissions:")
		fmt.Println()

		if len(perms) == 0 {
			perms = append(perms, "      No user specific permissions granted")
		}

		for _, p := range perms {
			fmt.Println(p)
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
