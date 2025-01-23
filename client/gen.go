// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/choria-io/go-choria/generators/client"
)

func generate(agent string, ddl string, pkg string) error {
	if ddl == "" {
		ddl = fmt.Sprintf("internal/fs/ddl/cache/agent/%s.json", agent)
	}

	if pkg == "" {
		pkg = agent + "client"
	}

	g := &client.Generator{
		DDLFile:     ddl,
		OutDir:      fmt.Sprintf("client/%sclient", agent),
		PackageName: pkg,
	}

	err := os.RemoveAll(g.OutDir)
	if err != nil {
		return err
	}

	err = os.Mkdir(g.OutDir, 0775)
	if err != nil {
		return err
	}

	err = g.GenerateClient()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	for _, agent := range []string{"rpcutil", "choria_util", "scout", "choria_provision", "choria_registry", "aaa_signer", "executor"} {
		err := generate(agent, "", "")
		if err != nil {
			panic(err)
		}
	}
}
