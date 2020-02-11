// +build ignore

package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
)

type Template struct {
	Name string
	Body string
}

type Templates []Template

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

var templateWrapperTemplate = `
package client

var templates = map[string]string{
{{- range . }}
	"{{ .Name }}": "{{ .Body }}",
{{- end }}
}
`

func goFmt(file string) error {
	c := exec.Command("go", "fmt", file)
	out, err := c.CombinedOutput()
	if err != nil {
		log.Printf("go fmt failed: %s", string(out))
	}

	return err
}

func main() {
	fmt.Println("Importing client code generation templates")

	tpath := path.Join("generators", "client", "templates")
	files, err := ioutil.ReadDir(tpath)
	panicIfErr(err)

	templates := Templates{}

	for _, file := range files {
		fname := strings.TrimSuffix(file.Name(), path.Ext(file.Name()))
		source := path.Join(tpath, file.Name())
		log.Printf("Generating %s from %s\n", fname, source)
		templ, err := ioutil.ReadFile(source)
		panicIfErr(err)

		templates = append(templates, Template{Name: fname, Body: base64.StdEncoding.EncodeToString(templ)})
	}

	t, err := template.New("templates").Parse(templateWrapperTemplate)
	panicIfErr(err)

	out, err := os.Create(path.Join("generators", "client", "templates.go"))
	panicIfErr(err)

	err = t.Execute(out, templates)
	panicIfErr(err)

	out.Close()
	err = goFmt(out.Name())
	panicIfErr(err)
}
