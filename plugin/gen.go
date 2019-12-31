// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/choria-io/go-choria/plugin"
)

var ctr = 1

func generateLoader(p *plugin.Plugin) (path string, err error) {
	fname := fmt.Sprintf("plugin_%0d_%s.go", ctr, p.Name)
	log.Printf("Generating loading code for plugin %s from %s into %s", p.Name, p.Repo, fname)

	ctr++

	f, err := os.Create(fname)
	if err != nil {
		return fname, fmt.Errorf("cannot create file %s: %s", fname, err)
	}
	defer f.Close()

	loader, err := p.Loader()
	if err != nil {
		return fname, fmt.Errorf("Could not create loader text: %s", err)
	}

	fmt.Fprint(f, loader)

	return fname, nil
}

func goFmt(file string) error {
	c := exec.Command("go", "fmt", file)
	out, err := c.CombinedOutput()
	if err != nil {
		log.Printf("go fmt failed: %s", string(out))
	}

	return err
}

func parseAndGenerate(file string) error {
	list, err := plugin.Load(file)
	if err != nil {
		return fmt.Errorf("could not load %s: %s", file, err)
	}

	for _, p := range list.Plugins {
		f, err := generateLoader(p)
		if err != nil {
			log.Printf("Could not generate loader for %s: %s", p.Name, err)
			return fmt.Errorf("some plugins failed to generate")
		}

		err = goFmt(f)
		if err != nil {
			log.Printf("Could not go fmt %s: %s", f, err)
		}
	}

	return err
}

func main() {
	success := true

	for _, file := range []string{"packager/plugins.yaml", "packager/user_plugins.yaml"} {
		if !fileExists(file) {
			continue
		}

		log.Printf("Generating plugin loaders from %s", file)

		err := parseAndGenerate(file)
		if err != nil {
			log.Printf("could not generate loaders from %s: %s", file, err)
			success = false
		}
	}

	if !success {
		os.Exit(1)
	}
}

func fileExists(f string) bool {
	_, err := os.Stat(f)

	return err == nil
}
