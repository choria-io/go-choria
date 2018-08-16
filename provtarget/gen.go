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

const ftempl = `// auto generated {{.Now}}
package main

import (
	ptp "{{.Repo}}"

	"github.com/choria-io/go-choria/provtarget"
)

func init() {
	provtarget.RegisterTargetResolver(ptp.Provider())
}
`

func (p provider) Now() string {
	return fmt.Sprintf("%s", time.Now())
}

func main() {
	if _, err := os.Stat("packager/provision_target_provider.yaml"); os.IsNotExist(err) {
		os.Exit(0)
	}

	j, err := ioutil.ReadFile("packager/provision_target_provider.yaml")
	if err != nil {
		log.Fatalf("Could not read provisioning target provider spec file packager/provision_target_provider.yaml: %s", err)
	}

	j, err = yaml.YAMLToJSON(j)
	if err != nil {
		log.Fatalf("Could not parse provisioning target provider spec file packager/provision_target_provider.yaml as YAML: %s", err)
	}

	prov := provider{}
	err = json.Unmarshal(j, &prov)
	if err != nil {
		log.Fatalf("Could not JSON parse converted YAML: %s", err)
	}

	templ := template.Must(template.New("provider").Parse(ftempl))

	fname := "provisioning_target_provider.go"

	log.Printf("Generating loading code for provisioning target provider %s from %s into %s", prov.Name, prov.Repo, fname)

	f, err := os.Create(fname)
	if err != nil {
		log.Fatalf("cannot create file %s: ", fname, err)
		return
	}
	defer f.Close()

	err = templ.Execute(f, prov)
	if err != nil {
		log.Println("executing template:", err)
	}
}
