package fs

import (
	"bytes"
	"embed"
	"strings"
	"text/template"

	"github.com/fatih/color"

	"github.com/choria-io/go-choria/internal/util"
)

//go:embed ddl
//go:embed client
//go:embed plugin
//go:embed misc
var FS embed.FS

type consoleRender interface {
	RenderConsole() ([]byte, error)
}

type mdRender interface {
	RenderMarkdown() ([]byte, error)
}

func ExecuteTemplate(file string, i interface{}, funcMap template.FuncMap) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	t := template.New(file)
	funcs := map[string]interface{}{
		"StringsJoin":    stringsJoin,
		"RenderConsole":  renderConsolePadded,
		"RenderMarkdown": renderMarkdown,
		"Bold":           boldString,
		"Title":          strings.Title,
	}

	for k, v := range funcMap {
		funcs[k] = v
	}

	t.Funcs(funcs)

	body, err := FS.ReadFile(file)
	if err != nil {
		return nil, err
	}

	p, err := t.Parse(string(body))
	if err != nil {
		return nil, err
	}

	err = p.Execute(buf, i)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func stringsJoin(s []string) string {
	return strings.Join(s, ", ")
}

func boldString(s string) string {
	return color.New(color.Bold).Sprintf(s)
}

func renderMarkdown(i mdRender) string {
	out, err := i.RenderMarkdown()
	if err != nil {
		panic(err)
	}

	return string(out)
}

func renderConsolePadded(i consoleRender, padding int) string {
	out, err := i.RenderConsole()
	if err != nil {
		panic(err)
	}

	return util.ParagraphPadding(string(out), padding)
}
