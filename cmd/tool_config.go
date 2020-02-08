package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/config"
)

type tConfigCommand struct {
	command

	key string
}

func (cc *tConfigCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		cc.cmd = tool.Cmd().Command("config", "Show documentation for a configuration item")
		cc.cmd.Arg("key", "The configuration keys to look up, supports regular expressions").Required().StringVar(&cc.key)
	}

	return nil
}

func (cc *tConfigCommand) Configure() (err error) {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return err
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	cfg.ApplyBuildSettings(bi)

	return err
}

func (cc *tConfigCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	keys, err := cfg.ConfigKeys(cc.key)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return fmt.Errorf("no configuration keys declared matching %q", cc.key)
	}

	for _, key := range keys {
		doc := cfg.DocForConfigKey(key)
		if doc == nil {
			continue
		}

		fmt.Printf("Configuration item: %q\n\n", doc.ConfigKey())
		fmt.Printf("      Description: %s\n", doc.Description())
		if doc.Deprecate() {
			fmt.Printf("       Deprecated: %t\n", doc.Deprecate())
		}
		if doc.URL() != "" {
			fmt.Printf("              URL: %s\n", doc.URL())
		}
		fmt.Printf("        Data Type: %s\n", doc.Type())
		if doc.Validation() != "" {
			fmt.Printf("       Validation: %s\n", doc.Validation())
		}
		if doc.Default() != "" {
			fmt.Printf("          Default: %s\n", doc.Default())
		}
		if doc.Environment() != "" {
			fmt.Printf("      Environment: %s\n", doc.Environment())
		}
		fmt.Printf("    Structure Key: %s\n", doc.StructKey())
		fmt.Println()
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tConfigCommand{})
}
