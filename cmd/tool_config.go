package cmd

import (
	"fmt"
	"sort"
	"sync"

	"github.com/fatih/color"

	"github.com/choria-io/go-choria/config"
)

type tConfigCommand struct {
	command

	key string
	list bool
}

func (cc *tConfigCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		cc.cmd = tool.Cmd().Command("config", "Show documentation for a configuration item")
		cc.cmd.Arg("key", "The configuration keys to look up, supports regular expressions").Required().StringVar(&cc.key)
		cc.cmd.Flag("list","Only list matching config keys").Short('l').BoolVar(&cc.list)
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

	sort.Strings(keys)

	if cc.list{
		for _, k := range keys {
			fmt.Println(k)
		}
		return nil
	}

	bold := color.New(color.Bold).SprintFunc()
	warn := color.New(color.FgHiRed, color.Bold).SprintFunc()

	for _, key := range keys {
		doc := cfg.DocForConfigKey(key)
		if doc == nil {
			continue
		}

		fmt.Printf("Configuration item: %s\n\n", bold(doc.ConfigKey()))
		fmt.Printf("      Description: %s\n", doc.Description())
		if doc.Deprecate() {
			fmt.Printf("       Deprecated: %s\n", warn("yes"))
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
