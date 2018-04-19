// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"encoding/json"

	"github.com/alecthomas/template"
	"github.com/ghodss/yaml"
)

type provider struct {
	Name string
	Repo string
}

type providers struct {
	Providers []provider
}

const ftempl = `// auto generated {{.Now}}
package main

import (
	ap "{{.Repo}}"

	"github.com/choria-io/go-choria/server"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.Info("Registering additional agent provider {{.Name}} from {{.Repo}}")
	server.RegisterAdditionalAgentProvider(&ap.Provider{})
}
`

func (p provider) Now() string {
	return fmt.Sprintf("%s", time.Now())
}

func main() {
	if _, err := os.Stat("packager/agent_providers.yaml"); os.IsNotExist(err) {
		os.Exit(0)
	}

	j, err := ioutil.ReadFile("packager/agent_providers.yaml")
	if err != nil {
		log.Fatalf("Could not read agents spec file packager/agent_providers.yaml: %s", err)
	}

	j, err = yaml.YAMLToJSON(j)
	if err != nil {
		log.Fatalf("Could not parse agents spec file packager/agent_providers.yaml as YAML: %s", err)
	}

	extra := providers{}
	err = json.Unmarshal(j, &extra)
	if err != nil {
		log.Fatalf("Could not JSON parse converted YAML: %s", err)
	}

	templ := template.Must(template.New("provider").Parse(ftempl))

	for _, provider := range extra.Providers {
		fname := fmt.Sprintf("additional_agent_provider_%s.go", provider.Name)

		log.Printf("Generating loading code for agent provider %s from %s into %s", provider.Name, provider.Repo, fname)

		f, err := os.Create(fname)
		if err != nil {
			log.Fatalf("cannot create file %s: ", fname, err)
			return
		}
		defer f.Close()

		err = templ.Execute(f, provider)
		if err != nil {
			log.Println("executing template:", err)
		}
	}
}
