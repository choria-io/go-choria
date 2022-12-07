//go:build ignore
// +build ignore

package main

import (
	"log"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/confkey"
	"github.com/choria-io/go-choria/internal/fs"
	"github.com/choria-io/go-choria/internal/util"
)

type configs struct {
	Keys    [][]string
	Docs    []*confkey.Doc
	Now     string
	Version string
}

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
		Docs:    []*confkey.Doc{},
		Now:     time.Now().UTC().Format(time.RFC822),
		Version: build.Version,
	}

	util.SliceGroups(keys, 2, func(grp []string) {
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

	templ, err := fs.FS.ReadFile("misc/config_doc.templ")
	if err != nil {
		return
	}

	t, err := template.New("templates").Funcs(funcs).Delims("<<", ">>").Parse(string(templ))
	if err != nil {
		panic(err)
	}

	outfile := "docs/content/configuration/_index.md"

	out, err := os.Create(path.Join(outfile))
	if err != nil {
		panic(err)
	}
	defer out.Close()

	err = t.Execute(out, docs)

	log.Println("Generated " + outfile)
}
