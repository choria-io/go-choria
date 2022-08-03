// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ddlresolver

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/fs"
)

// InternalCachedDDLResolver resolves DDLs against the internal cache directory
type InternalCachedDDLResolver struct{}

func (r *InternalCachedDDLResolver) String() string {
	return "Binary Cache DDL Resolver"
}

func (r *InternalCachedDDLResolver) DDL(ctx context.Context, kind string, name string, target any, fw inter.Framework) error {
	b, err := r.DDLBytes(ctx, kind, name, fw)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, target)
}

func (r *InternalCachedDDLResolver) DDLBytes(_ context.Context, kind string, name string, _ inter.Framework) ([]byte, error) {
	if kind != "agent" {
		return nil, fmt.Errorf("unsupported ddl type %q", kind)
	}

	return fs.FS.ReadFile(fmt.Sprintf("ddl/cache/agent/%s.json", name))
}

func (r *InternalCachedDDLResolver) DDLNames(_ context.Context, kind string, _ inter.Framework) ([]string, error) {
	if kind != "agent" {
		return nil, fmt.Errorf("unsupported ddl type %q", kind)
	}

	dir, err := fs.FS.ReadDir("ddl/cache/agent")
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, f := range dir {
		if f.IsDir() {
			continue
		}

		ext := filepath.Ext(f.Name())
		if ext != ".json" {
			continue
		}

		names = append(names, strings.TrimSuffix(f.Name(), ext))
	}

	sort.Strings(names)

	return names, nil
}
