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
	"strings"

	rc "github.com/choria-io/go-choria/client/choria_registryclient"
	"github.com/choria-io/go-choria/inter"
	iu "github.com/choria-io/go-choria/internal/util"
)

// RegistryDDLResolver resolves DDL via the Choria Registry
type RegistryDDLResolver struct{}

func (r *RegistryDDLResolver) String() string {
	return "Choria Registry DDL Resolver"
}

func (r *RegistryDDLResolver) DDL(ctx context.Context, kind string, name string, target any, fw inter.Framework) error {
	b, err := r.DDLBytes(ctx, kind, name, fw)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, target)
}

func (r *RegistryDDLResolver) findInCache(kind string, name string, fw inter.Framework) ([]byte, error) {
	cache := fw.Configuration().Choria.RegistryClientCache
	if !iu.FileIsDir(cache) {
		return nil, fmt.Errorf("no cache found")
	}

	cfile := filepath.Join(cache, kind, name+".json")
	if !iu.FileExist(cfile) {
		return nil, fmt.Errorf("not found in cache")
	}

	return os.ReadFile(cfile)
}

func (r *RegistryDDLResolver) storeInCache(kind string, name string, data []byte, fw inter.Framework) error {
	cache := fw.Configuration().Choria.RegistryClientCache
	targetDir := filepath.Join(cache, kind)
	if !iu.FileIsDir(targetDir) {
		err := os.MkdirAll(targetDir, 0700)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(filepath.Join(targetDir, name+".json"), data, 0644)
}

func (r *RegistryDDLResolver) DDLBytes(ctx context.Context, kind string, name string, fw inter.Framework) ([]byte, error) {
	if kind != "agent" {
		return nil, fmt.Errorf("unsupported ddl type %q", kind)
	}

	if fw.Configuration().Choria.RegistryClientCache == "" {
		return nil, fmt.Errorf("registry client is not enabled")
	}

	if fw.Configuration().InitiatedByServer {
		return nil, fmt.Errorf("servers cannot resolve DDLs via the registry")
	}

	cached, _ := r.findInCache(kind, name, fw)
	if cached != nil {
		return cached, nil
	}

	if fw.Configuration().RegistryCacheOnly {
		return nil, fmt.Errorf("registry client is operating in cache only mode")
	}

	client, err := rc.New(fw)
	if err != nil {
		return nil, err
	}

	res, err := client.Ddl(name, kind).Format("json").Do(ctx)
	if err != nil {
		return nil, err
	}

	if res.Stats().ResponsesCount() < 1 {
		return nil, fmt.Errorf("did not receive any response from the registry")
	}

	ddl := []byte{}
	logger := fw.Logger("registry")

	res.EachOutput(func(res *rc.DdlOutput) {
		logger.Infof("Resolved DDL via service host %s", res.ResultDetails().Sender())

		if !res.ResultDetails().OK() {
			err = fmt.Errorf("invalid response: %s", res.ResultDetails().StatusMessage())
			return
		}

		ddl = []byte(res.Ddl())
	})
	if err != nil {
		return nil, err
	}

	err = r.storeInCache(kind, name, ddl, fw)
	if err != nil {
		logger.Warnf("Could not save DDL for %s/%s in local cache: %s", kind, name, err)
	}

	return ddl, nil
}

func (r *RegistryDDLResolver) cacheDDLNames(kind string, fw inter.Framework) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(fw.Configuration().Choria.RegistryClientCache, kind))
	if err != nil {
		return nil, err
	}

	found := []string{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		found = append(found, strings.TrimSuffix(entry.Name(), ".json"))
	}

	return found, nil
}

func (r *RegistryDDLResolver) DDLNames(ctx context.Context, kind string, fw inter.Framework) ([]string, error) {
	if kind != "agent" {
		return nil, fmt.Errorf("unsupported ddl type %q", kind)
	}

	if fw.Configuration().Choria.RegistryClientCache == "" {
		return nil, fmt.Errorf("registry client is not enabled")
	}

	if fw.Configuration().InitiatedByServer {
		return nil, fmt.Errorf("servers cannot resolve DDLs via the registry")
	}

	if fw.Configuration().RegistryCacheOnly {
		return r.cacheDDLNames(kind, fw)
	}

	client, err := rc.New(fw)
	if err != nil {
		return nil, err
	}

	res, err := client.Names(kind).Do(ctx)
	if err != nil {
		return nil, err
	}

	names := []string{}
	if res.Stats().ResponsesCount() < 1 {
		return nil, fmt.Errorf("did not receive any response from the registry")
	}

	res.EachOutput(func(res *rc.NamesOutput) {
		fw.Logger("registry").Infof("Resolved DDL via service host %s", res.ResultDetails().Sender())

		if !res.ResultDetails().OK() {
			err = fmt.Errorf("invalid response: %s", res.ResultDetails().StatusMessage())
			return
		}

		for _, v := range res.Names() {
			name, ok := v.(string)
			if ok {
				names = append(names, name)
			}
		}
	})

	return names, err
}
