package cmd

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/provtarget"
	"github.com/choria-io/go-config"
	"github.com/choria-io/go-protocol/protocol"
	gnatsd "github.com/nats-io/nats-server/v2/server"
	"rsc.io/goversion/version"
)

type buildinfoCommand struct {
	command
}

func (b *buildinfoCommand) Setup() (err error) {
	b.cmd = cli.app.Command("buildinfo", "View build settings")

	return
}

func (b *buildinfoCommand) Configure() (err error) {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("Could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	cfg.ApplyBuildSettings(&build.Info{})

	return
}

func (b *buildinfoCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	fmt.Println("Choria build settings:")
	fmt.Println()
	fmt.Println("Build Data:")
	fmt.Printf("     Version: %s\n", build.Version)
	fmt.Printf("     Git SHA: %s\n", build.SHA)
	fmt.Printf("  Build Date: %s\n", build.BuildDate)
	fmt.Printf("     License: %s\n", build.License)
	fmt.Printf("  Go Version: %s\n", runtime.Version())
	fmt.Println()
	fmt.Println("Network Broker Settings:")
	fmt.Printf("       Maximum Network Clients: %d\n", build.MaxBrokerClients())
	fmt.Printf("  Embedded NATS Server Version: %s\n", gnatsd.VERSION)

	mutators := config.MutatorNames()
	fmt.Println()
	fmt.Println("Configuration Mutators:")
	if len(mutators) > 0 {
		for _, m := range mutators {
			fmt.Printf("\t%s\n", m)
		}
	} else {
		fmt.Printf("\tnone\n")
	}

	fmt.Println()
	fmt.Println("Server Settings:")
	if build.ProvisionBrokerURLs != "" {
		fmt.Printf("            Provisioning Brokers: %s\n", build.ProvisionBrokerURLs)
	}
	if build.ProvisionBrokerSRVDomain != "" {
		fmt.Printf("         Provisioning SRV Domain: %s\n", build.ProvisionBrokerSRVDomain)
	}
	fmt.Printf("           Provisioning JWT file: %s\n", build.ProvisionJWTFile)
	fmt.Printf("            Provisioning Default: %t\n", build.ProvisionDefault())
	fmt.Printf("                Provisioning TLS: %t\n", build.ProvisionSecurity())
	fmt.Printf("      Default Provisioning Agent: %t\n", build.ProvisionAgent == "true")
	fmt.Printf("  Provisioning Registration Data: %s\n", build.ProvisionRegistrationData)
	fmt.Printf("              Provisioning Facts: %s\n", build.ProvisionFacts)
	fmt.Printf("    Provisioning Target Resolver: %s\n", provtarget.Name())
	if build.ProvisionToken != "" {
		fmt.Printf("              Provisioning Token: set\n")
	} else {
		fmt.Printf("              Provisioning Token: not set\n")
	}
	fmt.Println()
	fmt.Println("Agent Providers:")

	for _, p := range build.AgentProviders {
		fmt.Printf("\t%s\n", p)
	}

	fmt.Println()
	fmt.Println("Security Defaults:")
	fmt.Printf("            TLS: %s\n", build.TLS)
	fmt.Printf("  x509 Security: %t\n", protocol.IsSecure())

	if build.TLS != "true" || !protocol.IsSecure() {
		fmt.Println()
		fmt.Println("NOTE: The security of this build is non standard, you might be running without adequate protocol level security.  Please ensure this is the build you intend to be using.")
	}

	printGoMods()

	return
}

func printGoMods() {
	binary, err := os.Executable()
	if err != nil {
		fmt.Printf("Could not read dependency information: %s\n", err)
		return
	}

	fver, err := version.ReadExe(binary)
	if err != nil {
		fmt.Printf("Could not read dependency information: %s\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Compile time module dependencies:")
	fmt.Println()

	if fver.ModuleInfo != "" {
		printModuleInfo(fver.ModuleInfo)
	} else {
		fmt.Println("No module dependencies found")
	}

}

func printModuleInfo(modinfo string) {
	var rows [][]string
	for _, line := range strings.Split(strings.TrimSpace(modinfo), "\n") {
		row := strings.Split(line, "\t")
		if row[0] != "dep" {
			continue
		}

		if len(row) > 3 {
			row = row[:3]
		}

		rows = append(rows, row[1:])
	}

	var max []int
	for _, row := range rows {
		for i, c := range row {
			n := utf8.RuneCountInString(c)
			if i >= len(max) {
				max = append(max, n)
			} else if max[i] < n {
				max[i] = n
			}
		}
	}

	b := bufio.NewWriter(os.Stdout)
	for _, row := range rows {
		b.WriteString("\t")
		for len(row) > 0 && row[len(row)-1] == "" {
			row = row[:len(row)-1]
		}
		for i, c := range row {
			b.WriteString(c)
			if i+1 < len(row) {
				for j := utf8.RuneCountInString(c); j < max[i]+2; j++ {
					b.WriteRune(' ')
				}
			}
		}
		b.WriteRune('\n')
	}
	b.Flush()
}

func init() {
	cli.commands = append(cli.commands, &buildinfoCommand{})
}
