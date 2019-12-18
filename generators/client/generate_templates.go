// +build ignore

package main

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"fmt"
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

func main() {
	fmt.Println("Importing client code generation templates")

	tpath :=path.Join("generators", "client", "templates")
	files, err := ioutil.ReadDir(tpath)
	panicIfErr(err)

	templates := Templates{}

	for _, file := range files {
		fname := strings.TrimSuffix(file.Name(), path.Ext(file.Name()))
		source := path.Join(tpath, file.Name())
		fmt.Printf(">> %s\n", source)
		templ, err := ioutil.ReadFile(source)
		panicIfErr(err)

		templates = append(templates, Template{Name: fname, Body: base64.StdEncoding.EncodeToString(templ)})
	}

	t, err := template.New("templates").Parse(templateWrapperTemplate)
	panicIfErr(err)

	out, err := os.Create(path.Join("generators", "client", "templates.go"))
	panicIfErr(err)
	defer out.Close()

	t.Execute(out, templates)
}
