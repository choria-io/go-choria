// +build ignore

package main

import (
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/confkey"
)

type configs struct {
	Keys [][]string
	Docs []*confkey.Doc
}

var templ = `# Choria Configuration Settings

This is a list of all known Configuration settings. This list is based on declared settings within the Choria Go code base and so will not cover 100% of settings - plugins can contribute their own settings which are note known at compile time.

## Data Types

A few special types are defined, the rest map to standard Go types

|Type|Description|
|----|-----------|
|comma_split|A comma separated list of strings, possibly with spaces between|
|duration|A duration such as "1h", "300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"|
|path_split|A list of paths split by a OS specific PATH separator|
|path_string|A path that can include "~" for the users home directory|
|strings|A space separated list of strings|
|title_string|A string that will be stored as a Title String|

## Index

| | |
|-|-|
{{- range $i, $k := .Keys }}
|[{{ index $k 0 }}](#{{ index $k 0 | gha }})|[{{ index $k 1 }}](#{{ index $k 1 | gha }})|
{{- end }}

{{ range .Docs }}
## {{ .ConfigKey }}

 * **Type:** {{ .Type }}
{{- if .URL }}
 * **Additional Information:** {{ .URL }}
{{- end }}
{{- if .Validation }}
 * **Validation:** {{ .Validation }}
{{- end }}
{{- if .Default }}
 * **Default Value:** {{ .Default }}
{{- end }}
{{- if .Environment }}
 * **Environment Variable:** {{ .Environment }}
{{- end }}
{{- if ne .Description "Undocumented" }}

{{ .Description }}{{ end }}
{{- if .Deprecate }}

**This setting is deprecated or already unused**
{{- end }}
{{ end }}
`

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cfg, err := config.NewDefaultConfig()
	panicIfErr(err)

	keys, err := cfg.ConfigKeys(".")
	panicIfErr(err)

	if len(keys) == 0 {
		panic("no configuration keys found")
	}

	docs := configs{
		Docs: []*confkey.Doc{},
	}

	choria.SliceGroups(keys, 2, func(grp []string) {
		docs.Keys = append(docs.Keys, grp)
	})

	for _, key := range keys {
		doc := cfg.DocForConfigKey(key)
		if doc == nil {
			continue
		}

		docs.Docs = append(docs.Docs, doc)
	}

	if len(docs.Docs) == 0 {
		panic("no documentation strings were generated")
	}

	funcs := template.FuncMap{
		"gha": func(s string) string {
			return strings.ReplaceAll(s, ".", "")
		},
	}

	t, err := template.New("templates").Funcs(funcs).Parse(templ)
	if err != nil {
		panic(err)
	}

	outfile := "CONFIGURATION.md"

	out, err := os.Create(path.Join(outfile))
	if err != nil {
		panic(err)
	}
	defer out.Close()

	err = t.Execute(out, docs)

	log.Println("Generated " + outfile)
}
