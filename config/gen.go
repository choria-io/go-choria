// +build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/fatih/structtag"
)

type keydoc struct {
	Key  string
	Help string
}

var templ = `
package config

var docStrings = map[string]string{
{{- range . }}
	{{ .Key | printf "%q" }}: {{ .Help | printf "%q" }},
{{- end }}
}
`

func parseDocString(d string) string {
	d = strings.TrimSpace(d)

	idx := strings.Index(d, "@doc")
	if idx == -1 {
		return d
	}

	d = d[idx+5:]
	d = strings.ReplaceAll(d, "\n", " ")

	return d
}

func parseGoFile(f string, structName string) (docs []keydoc, err error) {
	docs = []keydoc{}

	d, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ParseComments)
	if err != nil {
		return docs, err
	}

	var inserr error

	ast.Inspect(d, func(n ast.Node) bool {
		if inserr != nil {
			return false
		}

		switch t := n.(type) {
		case *ast.TypeSpec:
			return t.Name.Name == structName
		case *ast.StructType:
			for _, field := range t.Fields.List {
				tag := ""
				doc := ""

				if field.Tag != nil && field.Tag.Kind == token.STRING {
					tag = strings.TrimRight(strings.TrimLeft(field.Tag.Value, "`"), "`")
				}

				switch {
				case len(strings.TrimSpace(field.Doc.Text())) > 0:
					doc = field.Doc.Text()
				case len(strings.TrimSpace(field.Comment.Text())) > 0:
					doc = field.Comment.Text()
				}

				if strings.Contains(tag, "confkey") && doc != "" {
					tags, err := structtag.Parse(tag)
					if err != nil {
						inserr = err
						return false
					}

					key, err := tags.Get("confkey")
					if err != nil {
						inserr = err
						return false
					}

					docs = append(docs, keydoc{key.Value(), parseDocString(doc)})
				}
			}
		}

		return true
	})

	if inserr != nil {
		return docs, inserr
	}

	return docs, nil
}

func goFmt(file string) error {
	c := exec.Command("go", "fmt", file)
	out, err := c.CombinedOutput()
	if err != nil {
		log.Printf("go fmt failed: %s", string(out))
	}

	return err
}

func main() {
	docs := []keydoc{}

	log.Println("Generating configuration doc strings")
	cd, err := parseGoFile(filepath.Join("config", "config.go"), "Config")
	if err != nil {
		panic(err)
	}
	docs = append(docs, cd...)

	cd, err = parseGoFile(filepath.Join("config", "choria.go"), "ChoriaPluginConfig")
	if err != nil {
		panic(err)
	}
	docs = append(docs, cd...)

	if len(docs) == 0 {
		panic("no documentation strings were generated")
	}

	t, err := template.New("templates").Parse(templ)
	if err != nil {
		panic(err)
	}

	outfile := filepath.Join("config", "docstrings.go")

	out, err := os.Create(path.Join(outfile))
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(out, "	// auto generated at %s\n\n", time.Now())

	err = t.Execute(out, docs)
	out.Close()
	goFmt(outfile)

	log.Println("Generated config/docstrings.go")
}
