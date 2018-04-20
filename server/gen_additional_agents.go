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

type agent struct {
	Name string
	Repo string
}

type agents struct {
	Agents []agent
}

const ftempl = `// auto generated {{.Now}}
package main

import (
	"context"
	"fmt"

	aa "{{.Repo}}"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-choria/server"
	"github.com/sirupsen/logrus"
)

func init() {
	server.RegisterAdditionalAgent(func(ctx context.Context, mgr *agents.Manager, connector choria.InstanceConnector, log *logrus.Entry) error {
		log.Info("Registering additional agent {{.Name}} from {{.Repo}}")

		a, err := aa.New(mgr)
		if err != nil {
			return fmt.Errorf("Could not create {{.Name}} agent: %s", err)
		}

		mgr.RegisterAgent(ctx, "{{.Name}}", a, connector)

		return nil
	})
}
`

func (a agent) Now() string {
	return fmt.Sprintf("%s", time.Now())
}

func main() {
	if _, err := os.Stat("packager/agents.yaml"); os.IsNotExist(err) {
		os.Exit(0)
	}

	j, err := ioutil.ReadFile("packager/agents.yaml")
	if err != nil {
		log.Fatalf("Could not read agents spec file packager/agents.yaml: %s", err)
	}

	j, err = yaml.YAMLToJSON(j)
	if err != nil {
		log.Fatalf("Could not parse agents spec file packager/agents.yaml as YAML: %s", err)
	}

	extra := agents{}
	err = json.Unmarshal(j, &extra)
	if err != nil {
		log.Fatalf("Could not JSON parse converted YAML: %s", err)
	}

	templ := template.Must(template.New("agent").Parse(ftempl))

	for _, agent := range extra.Agents {
		fname := fmt.Sprintf("additional_agent_%s.go", agent.Name)

		log.Printf("Generating loading code for agent %s from %s into %s", agent.Name, agent.Repo, fname)

		f, err := os.Create(fname)
		if err != nil {
			log.Fatalf("cannot create file %s: ", fname, err)
			return
		}
		defer f.Close()

		err = templ.Execute(f, agent)
		if err != nil {
			log.Println("executing template:", err)
		}
	}
}
