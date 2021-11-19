// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/srvcache"
	"github.com/fatih/color"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
)

type tConfigCommand struct {
	command

	hideVals bool
	key      string
	list     bool
}

func (cc *tConfigCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		cc.cmd = tool.Cmd().Command("config", "Show documentation for a configuration item")
		cc.cmd.Arg("key", "The configuration keys to look up, supports regular expressions").StringVar(&cc.key)
		cc.cmd.Flag("list", "Only list matching config keys").Short('l').BoolVar(&cc.list)
	}

	return nil
}

func (cc *tConfigCommand) Configure() (err error) {
	err = commonConfigure()
	if err != nil {
		cfg, err = config.NewDefaultConfig()
		if err != nil {
			return err
		}
		cfg.Choria.SecurityProvider = "file"
		cc.hideVals = true
	}

	cfg.DisableSecurityProviderVerify = true

	return err
}

func (cc *tConfigCommand) renderServers(servers srvcache.Servers, err error) string {
	if err != nil {
		return err.Error()
	}

	return strings.Join(servers.Strings(), ", ")
}

func (cc *tConfigCommand) checkFileExist(f string) string {
	if util.FileExist(f) {
		return c.Colorize("green", "found")
	}
	return c.Colorize("green", "absent")
}

func (cc *tConfigCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if !cc.list {
		fmt.Println("This is the active Choria configuration as resolved from")
		fmt.Println("its configuration files and defaults.")
		if cc.key == "" {
			fmt.Println()
			fmt.Println("Details about configuration items can be seen by passing an")
			fmt.Println("extra argument that will act as a search term against known")
			fmt.Println("configuration items.")
		}
		fmt.Println()
		fmt.Printf("Configuration Files: \n\n")
		fmt.Printf("           User Config: %s\n", choria.UserConfig())
		if len(cfg.ParsedFiles) > 0 {
			for i, f := range cfg.ParsedFiles {
				if i == 0 {
					fmt.Printf("          Loaded Files: %s\n", f)
					continue
				}
				fmt.Printf("                        %s\n", f)
			}
		}

		fmt.Println()
		fmt.Printf("Security Configuration: \n\n")
		if cfg.Choria.ClientAnonTLS {
			fmt.Printf("     Security Provider: Using Anonymous TLS\n")
		} else {
			errs, _ := c.ValidateSecurity()
			fmt.Printf("     Security Provider: %s\n", strings.Title(c.SecurityProvider()))
			if len(errs) == 0 {
				fmt.Printf("        Valid Security: %s\n", c.Colorize("green", "yes"))
			} else {
				fmt.Printf("        Valid Security: %s\n", c.Colorize("red", "no"))
				for i, err := range errs {
					if i == 0 {
						fmt.Printf("       Security Errors: %s\n", err)
						continue
					}
					fmt.Printf("                        %s\n", err)
				}
			}
		}

		fmt.Printf("             Caller ID: %s\n", c.CallerID())
		if c.OverrideCertname() != "" {
			fmt.Printf("              Certname: %s\n", c.OverrideCertname())
		} else {
			fmt.Printf("              Certname: %s\n", c.Certname())
		}

		if c.Config.Choria.PKCS11DriverFile != "" {
			fmt.Printf("          PKC11 Driver: %s (%s)\n", c.Config.Choria.PKCS11DriverFile, cc.checkFileExist(c.Config.Choria.PKCS11DriverFile))
			fmt.Printf("           PKCS11 Slot: %d\n", c.Config.Choria.PKCS11Slot)
		}

		if c.Config.Choria.FileSecurityKey != "" {
			fmt.Printf("           Certificate: %s (%s)\n", c.Config.Choria.FileSecurityCertificate, cc.checkFileExist(c.Config.Choria.FileSecurityCertificate))
			fmt.Printf("                   Key: %s (%s)\n", c.Config.Choria.FileSecurityKey, cc.checkFileExist(c.Config.Choria.FileSecurityKey))
			fmt.Printf("                    CA: %s (%s)\n", c.Config.Choria.FileSecurityCA, cc.checkFileExist(c.Config.Choria.FileSecurityCA))
			fmt.Printf("                 Cache: %s (%s)\n", c.Config.Choria.FileSecurityCache, cc.checkFileExist(c.Config.Choria.FileSecurityCache))
		}

		if c.Config.Choria.RemoteSignerService {
			fmt.Printf("        Request Signer: Choria AAA Service\n")
		} else if c.Config.Choria.RemoteSignerURL != "" {
			fmt.Printf("        Request Signer: %s\n", c.Config.Choria.RemoteSignerURL)
			if c.Config.Choria.RemoteSignerSigningCertFile != "" {
				fmt.Printf("   Request Signer Cert: %s (%s)\n", c.Config.Choria.RemoteSignerSigningCertFile, cc.checkFileExist(c.Config.Choria.RemoteSignerSigningCertFile))
			}
		}
		if c.Config.Choria.RemoteSignerTokenFile != "" {
			fmt.Printf("            Token File: %s\n", c.Config.Choria.RemoteSignerTokenFile)
		}

		fmt.Println()
		fmt.Println("Connectivity:")
		fmt.Println()
		fmt.Printf("       Main Collective: %s\n", c.Config.MainCollective)
		fmt.Printf("           Collectives: %s\n", strings.Join(c.Config.Collectives, ", "))
		if c.Config.Choria.RegistryClientCache != "" {
			fmt.Printf("      Service Registry: %s\n", c.Config.Choria.RegistryClientCache)
		} else {
			fmt.Printf("      Service Registry: disabled\n")
		}
		fmt.Printf(" ")
		if c.Config.Choria.UseSRVRecords {
			fmt.Printf("           SRV Domain: %s\n", c.Config.Choria.SRVDomain)
		} else {
			fmt.Printf("           SRV Domain: not enabled\n")
		}
		if c.FacterCmd() != "" {
			fmt.Printf("        Facter Command: %s\n", c.FacterCmd())
			d, err := c.FacterDomain()
			if err != nil {
				fmt.Printf("         Facter Domain: %s\n", color.RedString(err.Error()))
			} else {
				fmt.Printf("         Facter Domain: %s\n", d)
			}
			fqdn, err := c.FacterFQDN()
			if err != nil {
				fmt.Printf("           Facter FQDN: %s\n", color.RedString(err.Error()))
			} else {
				fmt.Printf("           Facter FQDN: %s\n", fqdn)
			}
		}
		if c.Config.Choria.NatsNGS {
			fmt.Printf("           Synadia NGS: yes\n")
		}
		fmt.Printf("       PuppetDB Server: %s\n", cc.renderServers(c.PuppetDBServers()))
		fmt.Printf("        Choria Brokers: %s\n", cc.renderServers(c.MiddlewareServers()))
		if c.IsFederated() {
			fmt.Printf("    Federation Brokers: %s\n", cc.renderServers(c.FederationMiddlewareServers()))
			fmt.Printf("           Collectives: %s\n", strings.Join(c.FederationCollectives(), ", "))
		} else {
			fmt.Printf("    Federation Brokers: not federated\n")
		}

		fmt.Println()
		fmt.Println("Loaded Configuration:")
		fmt.Println()
		fmt.Println("  These settings were specifically loaded from configuration")
		fmt.Println("  files and do not include any defaults.")
		fmt.Println()
		util.DumpMapStrings(cfg.UnParsedOptions(), 2)
		fmt.Println()
	}

	if cc.key == "" && !cc.list {
		return nil
	}

	keys, err := cfg.ConfigKeys(cc.key)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return fmt.Errorf("no configuration keys declared matching %q", cc.key)
	}

	if cc.list {
		for _, k := range keys {
			fmt.Println(k)
		}
		return nil
	}

	bold := color.New(color.Bold).SprintFunc()
	warn := color.New(color.FgHiRed, color.Bold).SprintFunc()

	cols := 70
	colsstr := os.Getenv("COLUMNS")
	if colsstr != "" {
		cols, err = strconv.Atoi(colsstr)
		if err != nil {
			cols = 70
		}
		if cols > 100 {
			cols = 100
		}
	}

	for _, key := range keys {
		doc := cfg.DocForConfigKey(key)
		if doc == nil {
			continue
		}

		field := reflect.ValueOf(*cfg).FieldByName(doc.StructKey())
		if strings.HasPrefix(doc.StructKey(), "Choria.") {
			field = reflect.ValueOf(*cfg.Choria).FieldByName(strings.TrimPrefix(doc.StructKey(), "Choria."))
		}

		fmt.Printf("Configuration item: %s\n\n", bold(doc.ConfigKey()))
		if !cc.hideVals && !field.IsZero() {
			fmt.Printf("║        Value: %v\n", field)
		}
		if doc.Deprecate() {
			fmt.Printf("║   Deprecated: %s\n", warn("yes"))
		}
		if doc.URL() != "" {
			fmt.Printf("║          URL: %s\n", doc.URL())
		}
		fmt.Printf("║    Data Type: %s\n", doc.Type())
		if doc.Validation() != "" {
			fmt.Printf("║   Validation: %s\n", doc.Validation())
		}
		if doc.Default() != "" {
			fmt.Printf("║      Default: %s\n", doc.Default())
		}
		if doc.Environment() != "" {
			fmt.Printf("║  Environment: %s\n", doc.Environment())
		}
		fmt.Println("║")
		fmt.Println(wordWrap(doc.Description(), cols, "║ "))
		fmt.Println("╙─")
		fmt.Println()
	}

	return nil
}

func wordWrap(text string, lineWidth int, prefix string) (wrapped string) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return
	}
	wrapped = prefix + words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + prefix + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}
	return
}

func init() {
	cli.commands = append(cli.commands, &tConfigCommand{})
}
