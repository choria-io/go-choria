// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ddlresolver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choria-io/go-choria/inter"
)

// FileSystemDDLResolver resolves DDL in the lib directories configured in the framework
type FileSystemDDLResolver struct{}

func (f *FileSystemDDLResolver) String() string {
	return "File System DDL Resolver"
}

func (f *FileSystemDDLResolver) DDL(ctx context.Context, kind string, name string, target any, fw inter.Framework) error {
	b, err := f.DDLBytes(ctx, kind, name, fw)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, target)
}

func (f *FileSystemDDLResolver) DDLBytes(_ context.Context, kind string, name string, fw inter.Framework) ([]byte, error) {
	if kind != "agent" && kind != "data" {
		return nil, fmt.Errorf("unsupported ddl type %q", kind)
	}

	var (
		b   []byte
		err error
	)

	f.EachFile(kind, f.libDirs(fw), func(n, p string) bool {
		if n == name {
			b, err = os.ReadFile(p)
			return true
		}
		return false
	})
	if err != nil {
		return nil, fmt.Errorf("could not find DDL %s/%s: %s", kind, name, err)
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("could not find DDL %s/%s", kind, name)
	}

	return b, nil
}

func (f *FileSystemDDLResolver) DDLNames(_ context.Context, kind string, fw inter.Framework) ([]string, error) {
	if kind != "agent" && kind != "data" {
		return nil, fmt.Errorf("unsupported ddl type %q", kind)
	}

	names := []string{}
	f.EachFile(kind, f.libDirs(fw), func(n, _ string) bool {
		names = append(names, n)
		return false
	})

	sort.Strings(names)

	return names, nil
}

func (f *FileSystemDDLResolver) libDirs(fw inter.Framework) []string {
	return append(fw.Configuration().LibDir, fw.Configuration().Choria.RubyLibdir...)
}

func (f *FileSystemDDLResolver) EachFile(kind string, libdirs []string, cb func(name string, path string) (br bool)) {
	for _, dir := range libdirs {
		for _, n := range []string{"choria", "mcollective"} {
			filepath.Walk(filepath.Join(dir, n, kind), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				_, name := filepath.Split(path)
				extension := filepath.Ext(name)

				if extension != ".json" {
					return nil
				}

				cb(strings.TrimSuffix(name, extension), path)

				return nil
			})
		}
	}
}
