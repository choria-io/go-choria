package plugin

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/choria-io/go-choria/choria"
)

type generator struct {
	ctr int
}

func Generate() bool {
	g := generator{}

	return g.generate()
}

func (g *generator) generateLoader(p *Plugin) (path string, err error) {
	fname := fmt.Sprintf("plugin_%0d_%s.go", g.ctr, p.Name)
	log.Printf("Generating loading code for plugin %s from %s into %s", p.Name, p.Repo, fname)

	g.ctr++

	f, err := os.Create(fname)
	if err != nil {
		return fname, fmt.Errorf("cannot create file %s: %s", fname, err)
	}
	defer f.Close()

	loader, err := p.Loader()
	if err != nil {
		return fname, fmt.Errorf("could not create loader text: %s", err)
	}

	fmt.Fprint(f, loader)

	return fname, nil
}

func (g *generator) goFmt(file string) error {
	c := exec.Command("go", "fmt", file)
	out, err := c.CombinedOutput()
	if err != nil {
		log.Printf("go fmt failed: %s", string(out))
	}

	return err
}

func (g *generator) parseAndGenerate(file string) error {
	list, err := Load(file)
	if err != nil {
		return fmt.Errorf("could not load %s: %s", file, err)
	}

	for _, p := range list.Plugins {
		f, err := g.generateLoader(p)
		if err != nil {
			log.Printf("Could not generate loader for %s: %s", p.Name, err)
			return fmt.Errorf("some plugins failed to generate")
		}

		err = g.goFmt(f)
		if err != nil {
			log.Printf("Could not go fmt %s: %s", f, err)
		}
	}

	return err
}

func (g *generator) generate() bool {
	success := true

	for _, file := range []string{"packager/plugins.yaml", "packager/user_plugins.yaml"} {
		if !choria.FileExist(file) {
			continue
		}

		log.Printf("Generating plugin loaders from %s", file)

		err := g.parseAndGenerate(file)
		if err != nil {
			log.Printf("could not generate loaders from %s: %s", file, err)
			success = false
		}
	}

	return success
}
