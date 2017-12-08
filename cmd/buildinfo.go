package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/build"
)

type buildinfoCommand struct {
	command
}

func (b *buildinfoCommand) Setup() (err error) {
	b.cmd = cli.app.Command("buildinfo", "View build settings")

	return
}

func (b *buildinfoCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	fmt.Println("Choria build settings:")
	fmt.Println("")
	fmt.Println("Build Data:")
	fmt.Printf("     Version: %s\n", build.Version)
	fmt.Printf("     Git SHA: %s\n", build.SHA)
	fmt.Printf("  Build Date: %s\n", build.BuildDate)
	fmt.Printf("     License: %s\n", build.License)
	fmt.Println("")
	fmt.Println("Security Defaults:")
	fmt.Printf("            TLS: %s\n", build.TLS)
	fmt.Printf("  x509 Security: %s\n", build.Secure)

	if build.TLS != "true" || build.Secure != "true" {
		fmt.Println("")
		fmt.Println("NOTE: The security of this build is non standard, you might be running without adequate protocol level security.  Please ensure this is the build you intend to be using.")
	}

	return
}

func init() {
	cli.commands = append(cli.commands, &buildinfoCommand{})
}
