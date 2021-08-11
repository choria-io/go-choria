package cmd

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/provtarget"
	gnatsd "github.com/nats-io/nats-server/v2/server"
	"rsc.io/goversion/version"
)

type buildinfoCommand struct {
	command
	dependencies bool
}

func (b *buildinfoCommand) Setup() (err error) {
	b.cmd = cli.app.Command("buildinfo", "View build settings")
	b.cmd.Flag("dependencies", "Show dependencies used to build the binary").Short('D').BoolVar(&b.dependencies)

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

	c.ConfigureProvisioning()

	fmt.Println("Choria build settings:")
	fmt.Println()
	fmt.Println("Build Data:")
	fmt.Printf("     Version: %s\n", bi.Version())
	fmt.Printf("     Git SHA: %s\n", bi.SHA())
	fmt.Printf("  Build Date: %s\n", bi.BuildDate())
	fmt.Printf("     License: %s\n", bi.License())
	fmt.Printf("  Go Version: %s\n", runtime.Version())
	fmt.Println()
	fmt.Println("Protocol Defaults and Settings:")
	fmt.Printf("     Default Collectives: %v\n", strings.Join(bi.DefaultCollectives(), ", "))
	fmt.Printf("  Client Identity Suffix: %s\n", bi.ClientIdentitySuffix())
	fmt.Println()
	fmt.Println("Network Broker Settings:")
	fmt.Printf("       Maximum Network Clients: %d\n", bi.MaxBrokerClients())
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
	fmt.Printf("            Provisioning Default: %t\n", bi.ProvisionDefault())
	fmt.Printf("                Provisioning TLS: %t\n", bi.ProvisionSecurity())
	fmt.Printf("    Provisioning Target Resolver: %s\n", provtarget.Name())
	fmt.Printf("      Default Provisioning Agent: %t\n", bi.ProvisionAgent())
	if bi.ProvisionToken() != "" {
		fmt.Printf("              Provisioning Token: *****\n")
	} else {
		fmt.Printf("              Provisioning Token: not set\n")
	}
	if bi.ProvisionBrokerURLs() != "" {
		fmt.Printf("            Provisioning Brokers: %s\n", bi.ProvisionBrokerURLs())
	}
	if bi.ProvisionBrokerSRVDomain() != "" {
		fmt.Printf("         Provisioning SRV Domain: %s\n", bi.ProvisionBrokerSRVDomain())
	}
	if bi.ProvisionJWTFile() != "" {
		fmt.Printf("           Provisioning JWT file: %s\n", bi.ProvisionJWTFile())
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

	fmt.Println()
	fmt.Println("Security Defaults:")
	fmt.Printf("            TLS: %t\n", bi.HasTLS())
	fmt.Printf("  x509 Security: %t\n", protocol.IsSecure())

	if !bi.HasTLS() || !protocol.IsSecure() {
		fmt.Println()
		fmt.Println("NOTE: The security of this build is not standard, you might be running without adequate protocol level security.  Please ensure this is the build you intend to be using.")
	}

	fmt.Println()
	fmt.Println("Agent Providers:")

	for _, p := range bi.AgentProviders() {
		fmt.Printf("  %s\n", p)
	}

	fmt.Println()
	fmt.Println("Data Providers:")

	for _, p := range bi.DataProviders() {
		fmt.Printf("  %s\n", p)
	}

	fmt.Println()
	fmt.Println("Autonomous Agent Watchers:")

	for _, p := range bi.MachineWatchers() {
		fmt.Printf("  %s\n", p)
	}

	if b.dependencies {
		b.printGoMods()
	}

	return
}

func (b *buildinfoCommand) printGoMods() {
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
		b.printModuleInfo(fver.ModuleInfo)
	} else {
		fmt.Println("No module dependencies found")
	}

}

func (b *buildinfoCommand) printModuleInfo(modinfo string) {
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

	out := bufio.NewWriter(os.Stdout)
	for _, row := range rows {
		out.WriteString("\t")
		for len(row) > 0 && row[len(row)-1] == "" {
			row = row[:len(row)-1]
		}
		for i, c := range row {
			out.WriteString(c)
			if i+1 < len(row) {
				for j := utf8.RuneCountInString(c); j < max[i]+2; j++ {
					out.WriteRune(' ')
				}
			}
		}
		out.WriteRune('\n')
	}
	out.Flush()
}

func init() {
	cli.commands = append(cli.commands, &buildinfoCommand{})
}
