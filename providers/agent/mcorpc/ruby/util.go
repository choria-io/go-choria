// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ruby

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/choria-io/go-choria/internal/util"
	agentddl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

func (p *Provider) loadAgents(libdirs []string) {
	p.eachAgent(libdirs, func(a *agentddl.DDL) {
		p.agents = append(p.agents, a)
	})
}

func (p *Provider) eachAgent(libdirs []string, cb func(ddl *agentddl.DDL)) {
	for _, dir := range libdirs {
		agentsdir := filepath.Join(dir, "mcollective", "agent")

		p.log.Debugf("Attempting to load Ruby agents from %s", agentsdir)

		err := filepath.Walk(agentsdir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			fname := info.Name()
			extension := filepath.Ext(fname)
			name := strings.TrimSuffix(fname, extension)

			if extension != ".json" {
				return nil
			}

			bpath := strings.TrimSuffix(path, extension)
			rbfile := bpath + ".rb"

			if !util.FileExist(rbfile) || util.FileIsDir(rbfile) {
				return nil
			}

			if !shouldLoadAgent(name) {
				p.log.Warnf("Ruby agents are not allowed to supply an agent called '%s', skipping", name)
				return nil
			}

			p.log.Debugf("Attempting to load %s as an agent DDL", path)
			ddl, err := agentddl.New(path)
			if err != nil {
				p.log.Errorf("Could not load ruby agent DDL %s: %s", path, err)
				return nil
			}

			if ddl.Metadata.Provider == "" || ddl.Metadata.Provider == "ruby" {
				cb(ddl)
			}

			return nil
		})

		if err != nil {
			p.log.Errorf("Could not find agents in %s: %s", dir, err)
		}
	}
}

func shouldLoadAgent(name string) bool {
	for _, a := range denylist {
		if a == name {
			return false
		}
	}

	return true
}
