// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"os"
	"text/template"
)

func ExecuteTemplateFile(file string, data any, funcs template.FuncMap) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	t := template.New(file)
	t.Funcs(funcs)

	body, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	p, err := t.Parse(string(body))
	if err != nil {
		return nil, err
	}

	err = p.Execute(buf, data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
