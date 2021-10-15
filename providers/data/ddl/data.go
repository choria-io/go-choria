// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ddl

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
)

// Metadata describes an agent at a high level and is required for any agent
type Metadata struct {
	License     string `json:"license"`
	Author      string `json:"author"`
	Timeout     int    `json:"timeout"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Provider    string `json:"provider,omitempty"`
}

type DDL struct {
	Schema   string                        `json:"$schema"`
	Metadata Metadata                      `json:"metadata"`
	Query    *common.InputItem             `json:"query"`
	Output   map[string]*common.OutputItem `json:"output"`

	SourceLocation string `json:"-"`

	sync.Mutex
}

// New creates a new DDL from a JSON file
func New(file string) (*DDL, error) {
	ddl := &DDL{
		SourceLocation: file,
	}

	dat, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not load DDL data: %s", err)
	}

	err = json.Unmarshal(dat, ddl)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON data in %s: %s", file, err)
	}

	return ddl, nil
}

func Find(plugin string, libdirs []string) (ddl *DDL, err error) {
	EachFile(libdirs, func(n string, f string) bool {
		if n == plugin {
			ddl, err = New(f)
			return true
		}

		return false
	})

	if err != nil {
		return nil, fmt.Errorf("could not load data plugin %s: %s", plugin, err)
	}

	if ddl == nil {
		return nil, fmt.Errorf("could not find DDL file for %s", plugin)
	}

	return ddl, nil
}

// EachFile calls cb with a path to every found data DDL, stops looking when br is true
func EachFile(libdirs []string, cb func(name string, path string) (br bool)) {
	common.EachFile("data", libdirs, cb)
}

// Timeout is the timeout for this data plugin
func (d *DDL) Timeout() time.Duration {
	if d.Metadata.Timeout == 0 {
		return 10 * time.Second
	}

	return time.Second * time.Duration(d.Metadata.Timeout)
}
