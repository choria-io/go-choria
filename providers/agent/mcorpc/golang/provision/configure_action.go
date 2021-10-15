// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/lifecycle"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/sirupsen/logrus"
)

type ConfigureRequest struct {
	Token          string            `json:"token"`
	Configuration  string            `json:"config"`
	Key            string            `json:"key"`
	Certificate    string            `json:"certificate"`
	CA             string            `json:"ca"`
	SSLDir         string            `json:"ssldir"`
	ECDHPublic     string            `json:"ecdh_public"`
	ActionPolicies map[string]string `json:"action_policies"`
	OPAPolicies    map[string]string `json:"opa_policies"`
}

func configureAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() {
		abort("Cannot reconfigure a server that is not in provisioning mode", reply)
		return
	}

	if agent.Config.ConfigFile == "" {
		abort("Cannot determine the configuration file to manage", reply)
		return
	}

	args := &ConfigureRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	if !checkToken(args.Token, reply) {
		return
	}

	if len(args.Configuration) == 0 {
		abort("Did not receive any configuration to write, cannot write a empty configuration file", reply)
		return
	}

	if len(args.Key) != 0 && len(args.ECDHPublic) == 0 {
		abort("EDCH Public Key not supplied while providing a private key", reply)
		return
	}

	settings := make(map[string]string)
	err := json.Unmarshal([]byte(args.Configuration), &settings)
	if err != nil {
		abort(fmt.Sprintf("Could not decode configuration data: %s", err), reply)
		return
	}

	err = writeTLS(args, agent)
	if err != nil {
		abort(fmt.Sprintf("TLS Setup Failed: %s", err), reply)
		return
	}

	lines, err := writeConfig(settings, req, agent.Config, agent.Log)
	if err != nil {
		abort(fmt.Sprintf("Could not write config: %s", err), reply)
		return
	}

	err = writeActionPolicies(agent.Config, args, agent)
	if err != nil {
		abort(fmt.Sprintf("Could not save Action Policies: %s", err), reply)
		return
	}

	err = writeOPAPolicies(agent.Config, args, agent)
	if err != nil {
		abort(fmt.Sprintf("Could not save Open Policy Agent Policies: %s", err), reply)
		return
	}

	err = agent.ServerInfoSource.NewEvent(lifecycle.Provisioned)
	if err != nil {
		agent.Log.Errorf("Could not publish provisioned event: %s", err)
	}

	reply.Data = Reply{fmt.Sprintf("Wrote %d lines to %s", lines, agent.Config.ConfigFile)}
}

func writePolicies(targetDir string, policies map[string]string, log *logrus.Entry) error {
	if util.FileIsDir(targetDir) {
		err := os.RemoveAll(targetDir)
		if err != nil {
			return fmt.Errorf("could not remove existing policies")
		}
	}

	err := os.MkdirAll(targetDir, 0700)
	if err != nil {
		return err
	}

	for f, p := range policies {
		if f == "" || p == "" {
			continue
		}

		target := filepath.Join(targetDir, f)
		err = os.WriteFile(target, []byte(p), 0400)
		if err != nil {
			return fmt.Errorf("saving policy %v failed: %v", f, err)
		}
		log.Infof("Wrote policy to %s", target)
	}

	return nil
}

func writeOPAPolicies(cfg *config.Config, args *ConfigureRequest, agent *mcorpc.Agent) error {
	if len(args.OPAPolicies) == 0 {
		return nil
	}

	if cfg.ConfigFile == "" {
		return fmt.Errorf("no configuration path configured")
	}

	for k := range args.OPAPolicies {
		if !strings.HasSuffix(k, ".rego") {
			return fmt.Errorf("expected %q to have .rego extension", k)
		}
	}

	return writePolicies(filepath.Join(filepath.Dir(cfg.ConfigFile), "policies", "rego"), args.OPAPolicies, agent.Log)
}

func writeActionPolicies(cfg *config.Config, args *ConfigureRequest, agent *mcorpc.Agent) error {
	if len(args.ActionPolicies) == 0 {
		return nil
	}

	if cfg.ConfigFile == "" {
		return fmt.Errorf("no configuration path configured")
	}

	for k := range args.ActionPolicies {
		if !strings.HasSuffix(k, ".policy") {
			return fmt.Errorf("expected %q to have .policy extension", k)
		}
	}

	return writePolicies(filepath.Join(filepath.Dir(cfg.ConfigFile), "policies"), args.ActionPolicies, agent.Log)
}

func writeTLS(args *ConfigureRequest, agent *mcorpc.Agent) error {
	if args.Certificate != "" && args.SSLDir != "" && args.CA != "" {
		err := os.MkdirAll(args.SSLDir, 0700)
		if err != nil {
			return fmt.Errorf("could not create ssl directory %s: %s", args.SSLDir, err)
		}

		target := filepath.Join(args.SSLDir, "certificate.pem")
		err = os.WriteFile(target, []byte(args.Certificate), 0644)
		if err != nil {
			return fmt.Errorf("could not write Certificate to %s: %s", target, err)
		}

		target = filepath.Join(args.SSLDir, "ca.pem")
		err = os.WriteFile(target, []byte(args.CA), 0644)
		if err != nil {
			return fmt.Errorf("could not write CA to %s: %s", target, err)
		}

		if args.Key != "" {
			agent.Log.Warnf("Received a PRIVATE KEY over the network")

			priKey, err := decryptPrivateKey(args.Key, args.ECDHPublic)
			if err != nil {
				return fmt.Errorf("could not decrypt private key: %s", err)
			}

			target = filepath.Join(args.SSLDir, "private.pem")
			err = os.WriteFile(target, priKey, 0600)
			if err != nil {
				return fmt.Errorf("could not write KEY to %s: %s", target, err)
			}

			csrFile := filepath.Join(args.SSLDir, "csr.pem")
			if util.FileExist(csrFile) {
				agent.Log.Warnf("A PRIVATE KEY was received from the provisioner, removing CSR %s", csrFile)
				err = os.Remove(csrFile)
				if err != nil {
					agent.Log.Errorf("A PRIVATE KEY was received from the provisioner, could not remove CSR %s: %s", csrFile, err)
				}
			}
		}
	}

	return nil
}

func writeConfig(settings map[string]string, req *mcorpc.Request, cfg *config.Config, log *logrus.Entry) (int, error) {
	cfile := cfg.ConfigFile

	_, err := os.Stat(cfile)
	if err == nil {
		cfile, err = filepath.EvalSymlinks(cfile)
		if err != nil {
			return 0, fmt.Errorf("cannot determine full path to config file %s: %s", cfile, err)
		}
	}

	log.Warnf("Rewriting configuration file %s in request %s from %s (%s)", cfile, req.RequestID, req.CallerID, req.SenderID)

	cdir := filepath.Dir(cfile)

	tmpfile, err := os.CreateTemp(cdir, "provision")
	if err != nil {
		return 0, fmt.Errorf("cannot create a temp file in %s: %s", cdir, err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	_, err = fmt.Fprintf(tmpfile, "# configuration file written in request %s from %s (%s) at %s\n", req.RequestID, req.CallerID, req.SenderID, time.Now())
	if err != nil {
		return 0, fmt.Errorf("could not write to temp file %s: %s", tmpfile.Name(), err)
	}

	written := 1

	for k, v := range settings {
		log.Infof("Adding configuration: %s = %s", k, v)

		_, err := fmt.Fprintf(tmpfile, "%s=%s\n", k, v)
		if err != nil {
			return 0, fmt.Errorf("could not write to temp file %s: %s", tmpfile.Name(), err)
		}

		written++
	}

	err = tmpfile.Close()
	if err != nil {
		return 0, fmt.Errorf("could not close temp file %s: %s", tmpfile.Name(), err)
	}

	_, err = config.NewConfig(tmpfile.Name())
	if err != nil {
		return 0, fmt.Errorf("generated configuration could not be parsed: %s", err)
	}

	err = os.Rename(tmpfile.Name(), cfile)
	if err != nil {
		return 0, fmt.Errorf("could not rename temp file %s to %s: %s", tmpfile.Name(), cfile, err)
	}

	return written, nil
}
