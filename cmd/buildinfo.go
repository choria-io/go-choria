// Copyright (c) 2017-2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"github.com/choria-io/go-choria/config"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/provtarget"
	gnatsd "github.com/nats-io/nats-server/v2/server"
	"runtime"
	rd "runtime/debug"
	"sort"
	"strings"
	"sync"
)

type buildinfoCommand struct {
	command
	dependencies bool
}

func (b *buildinfoCommand) Setup() (err error) {
	b.cmd = cli.app.Command("buildinfo", "Build Settings and Configuration")
	b.cmd.Flag("dependencies", "Show dependencies used to build the binary").Short('D').UnNegatableBoolVar(&b.dependencies)

	return
}

func (b *buildinfoCommand) Configure() (err error) {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	cfg.ApplyBuildSettings(bi)

	return
}

func (b *buildinfoCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	c.ConfigureProvisioning(ctx)

	fmt.Println("Choria build settings:")
	fmt.Println()
	fmt.Println("Build Data:")
	fmt.Println()
	fmt.Printf("     Version: %s\n", bi.Version())
	fmt.Printf("     Git SHA: %s\n", bi.SHA())
	fmt.Printf("  Build Date: %s\n", bi.BuildDate())
	fmt.Printf("     License: %s\n", bi.License())
	fmt.Printf("  Go Version: %s\n", runtime.Version())
	fmt.Println()
	fmt.Println("Protocol Defaults and Settings:")
	fmt.Println()
	fmt.Printf("     Default Collectives: %v\n", strings.Join(bi.DefaultCollectives(), ", "))
	fmt.Printf("  Client Identity Suffix: %s\n", bi.ClientIdentitySuffix())
	fmt.Println()
	fmt.Println("Network Broker Settings:")
	fmt.Println()
	fmt.Printf("       Maximum Network Clients: %d\n", bi.MaxBrokerClients())
	fmt.Printf("  Embedded NATS Server Version: %s\n", gnatsd.VERSION)

	mutators := config.MutatorNames()
	if len(mutators) > 0 {
		fmt.Println()
		fmt.Println("Configuration Mutators:")
		fmt.Println()

		for _, m := range mutators {
			fmt.Printf("\t%s\n", m)
		}
	}

	fmt.Println()
	fmt.Println("Server Settings:")
	fmt.Println()
	fmt.Printf("    Provisioning Target Resolver: %s\n", provtarget.Name())
	fmt.Printf("           Supports Provisioning: %t\n", bi.SupportsProvisioning())
	if bi.ProvisionJWTFile() != "" {
		fmt.Printf("           Provisioning JWT file: %s\n", bi.ProvisionJWTFile())
	}
	if bi.SupportsProvisioning() {
		if bi.ProvisionToken() != "" {
			fmt.Printf("              Provisioning Token: *****\n")
		} else {
			fmt.Printf("              Provisioning Token: not set\n")
		}
		fmt.Printf("            Provisioning Default: %t\n", bi.ProvisionDefault())
		fmt.Printf("                Provisioning TLS: %t\n", bi.ProvisionSecurity())
		fmt.Printf("      Default Provisioning Agent: %t\n", bi.ProvisionAgent())
		if bi.ProvisionBrokerURLs() != "" {
			fmt.Printf("            Provisioning Brokers: %s\n", bi.ProvisionBrokerURLs())
		}
		if bi.ProvisionBrokerSRVDomain() != "" {
			fmt.Printf("         Provisioning SRV Domain: %s\n", bi.ProvisionBrokerSRVDomain())
		}
		if bi.ProvisionRegistrationData() != "" {
			fmt.Printf("  Provisioning Registration Data: %s\n", bi.ProvisionRegistrationData())
		}
		if bi.ProvisionFacts() != "" {
			fmt.Printf("              Provisioning Facts: %s\n", bi.ProvisionFacts())
		}
		if bi.ProvisioningBrokerUsername() != "" {
			fmt.Printf("    Provisioning Broker Username: %s\n", bi.ProvisioningBrokerUsername())
		}
		if bi.ProvisioningBrokerUsername() != "" {
			fmt.Println("    Provisioning Broker Password: ******")
		}
		if bi.ProvisionUsingVersion2() {
			fmt.Println("  Provisioning Using Protocol v2: true")
		}
		if bi.ProvisionAllowServerUpdate() {
			fmt.Println("   Provisioning Version Upgrades: true")
		}
	}

	fmt.Println()
	fmt.Println("Security Defaults:")
	fmt.Println()
	fmt.Printf("            TLS: %t\n", bi.HasTLS())
	fmt.Printf("  x509 Security: %t\n", protocol.IsSecure())

	if !bi.HasTLS() || !protocol.IsSecure() {
		fmt.Println()
		fmt.Println("NOTE: The security of this build is not standard, you might be running without adequate protocol level security.  Please ensure this is the build you intend to be using.")
	}

	fmt.Println()
	fmt.Println("Agent Providers:")
	fmt.Println()

	for _, p := range bi.AgentProviders() {
		fmt.Printf("  %s\n", p)
	}

	data := bi.DataProviders()
	if len(data) > 0 {
		fmt.Println()
		fmt.Println("Data Providers:")
		fmt.Println()

		for _, p := range data {
			fmt.Printf("  %s\n", p)
		}
	}

	machines := bi.Machines()
	if len(machines) > 0 {
		fmt.Println()
		fmt.Println("Autonomous Agents:")
		fmt.Println()

		for _, p := range machines {
			fmt.Printf("  %s\n", p)
		}
	}

	watchers := bi.MachineWatchers()
	if len(watchers) > 0 {
		fmt.Println()
		fmt.Println("Autonomous Agent Watchers:")
		fmt.Println()

		for _, p := range watchers {
			fmt.Printf("  %s\n", p)
		}
	}

	if b.dependencies {
		b.printGoMods()
	}

	return
}

func (b *buildinfoCommand) printGoMods() {
	nfo, ok := rd.ReadBuildInfo()
	if !ok {
		fmt.Println("Could not read dependency information")
		return
	}

	fmt.Println()
	fmt.Println("Compile time module dependencies:")
	fmt.Println()

	if len(nfo.Deps) == 0 {
		fmt.Println("No module dependencies found")
		return
	}

	mods := []string{}
	versions := map[string]string{}
	for _, mod := range nfo.Deps {
		mods = append(mods, mod.Path)
		versions[mod.Path] = mod.Version
	}

	longest := iu.LongestString(mods, 50)
	sort.Strings(mods)

	format := fmt.Sprintf("  %%%ds %%s\n", longest)
	for _, mod := range mods {
		fmt.Printf(format, mod, versions[mod])
	}
}

func init() {
	cli.commands = append(cli.commands, &buildinfoCommand{})
}
