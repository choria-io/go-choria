// +build ignore

package main

import (
	"log"
	"os"
	"path"
	"text/template"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/confkey"
)

var templ = `# Choria Configuration Settings

This is a list of all known Configuration settings. This list is based on declared settings within the Choria Go code base and so will not cover 100% of settings - plugins can contribute their own settings.

Some emoji are used: 

 * :spider_web: Deprecated setting
 * :notebook: Additional information

|Key|Description|
|---|-----------|
{{- range . }}
|{{ .ConfigKey }} {{ if .Deprecate }}:spider_web:{{ end }}{{ if .URL }}[:notebook:]({{ .URL }}){{ end }}|{{ .Description }}|
{{- end }}
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

	docs := []*confkey.Doc{}

	for _, key := range keys {
		doc := cfg.DocForConfigKey(key)
		if doc == nil {
			continue
		}

		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		panic("no documentation strings were generated")
	}

	t, err := template.New("templates").Parse(templ)
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
