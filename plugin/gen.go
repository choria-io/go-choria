// +build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/choria-io/go-choria/plugin"
)

var ctr = 1

func generateLoader(p *plugin.Plugin) error {
	fname := fmt.Sprintf("plugin_%0d_%s.go", ctr, p.Name)
	log.Printf("Generating loading code for plugin %s from %s into %s", p.Name, p.Repo, fname)

	ctr++

	f, err := os.Create(fname)
	if err != nil {
		return fmt.Errorf("cannot create file %s: ", fname, err)
	}
	defer f.Close()

	loader, err := p.Loader()
	if err != nil {
		return fmt.Errorf("Could not create loader text: %s", err)
	}

	fmt.Fprint(f, loader)

	return nil
}

func parseAndGenerate(file string) error {
	list, err := plugin.Load(file)
	if err != nil {
		return fmt.Errorf("could not load %s: %s", file, err)
	}

	for _, p := range list.Plugins {
		err := generateLoader(p)
		if err != nil {
			log.Printf("Could not generate loader for %s: %s", p.Name, err)
			err = fmt.Errorf("some plugins failed to generate")
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
