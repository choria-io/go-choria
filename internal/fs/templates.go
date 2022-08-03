// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package fs

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/choria-io/go-choria/internal/util"
)

type consoleRender interface {
	RenderConsole() ([]byte, error)
}

type mdRender interface {
	RenderMarkdown() ([]byte, error)
}

func ExecuteTemplate(file string, i any, funcMap template.FuncMap) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	t := template.New(file)
	funcs := map[string]any{
		"StringsJoin":    stringsJoin,
		"RenderConsole":  renderConsolePadded,
		"RenderMarkdown": renderMarkdown,
		"MarkdownEscape": markdownEscape,
		"Bold":           boldString,
		"Title":          cases.Title(language.AmericanEnglish).String,
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

// use to escape within backticks, not for general full md escape, we mainly want to avoid escaping backticks and tables
func markdownEscape(s string) string {
	escaped := s
	for _, c := range strings.Fields("` |") {
		escaped = strings.Replace(escaped, c, "\\"+c, -1)
	}

	return escaped
}
