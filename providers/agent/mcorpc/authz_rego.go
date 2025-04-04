// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package mcorpc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/server/agents"
	"path/filepath"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/opa"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/sirupsen/logrus"
)

type regoPolicy struct {
	cfg *config.Config
	req *Request
	fw  inter.Framework
	si  agents.ServerInfoSource
	log *logrus.Entry
}

func regoPolicyAuthorize(req *Request, fw inter.Framework, si agents.ServerInfoSource, cfg *config.Config, log *logrus.Entry) (bool, error) {
	logger := log.WithFields(logrus.Fields{
		"authorizer": "regoPolicy",
		"agent":      req.Agent,
		"request":    req.RequestID,
	})

	authz := &regoPolicy{
		cfg: cfg,
		req: req,
		si:  si,
		fw:  fw,
		log: logger,
	}

	return authz.authorize()
}

func (r *regoPolicy) authorize() (bool, error) {
	policyFile, err := r.lookupPolicyFile()
	if err != nil {
		return false, err
	}

	if policyFile == "" {
		return false, fmt.Errorf("policy file could not be found")
	}

	eopts := []opa.Option{
		opa.Logger(r.log),
		opa.File(policyFile),
	}

	if r.log.Logger.GetLevel() == logrus.DebugLevel || r.enableTracing() {
		r.log.Debugf("regoInputs: %v", r.regoInputs())
		eopts = append(eopts, opa.Trace())
	}

	evaluator, err := opa.New("io.choria.mcorpc.authpolicy", "data.io.choria.mcorpc.authpolicy.allow", eopts...)
	if err != nil {
		return false, err
	}

	inputs := r.regoInputs()
	allowed, err := evaluator.Evaluate(context.Background(), inputs)
	switch err := err.(type) {
	case nil:
		break
	case ast.Errors:
		for _, e := range err {
			r.log.Info("code: ", e.Code)
			r.log.Info("row: ", e.Location.Row)
			r.log.Info("filename: ", policyFile)
		}
		return false, err
	default:
		return false, err
	}

	return allowed, nil
}

var overRideRegoName string

func (r *regoPolicy) lookupPolicyFile() (string, error) {
	dir := filepath.Join(filepath.Dir(r.cfg.ConfigFile), "policies", "rego")

	regoPolicy := filepath.Join(dir, r.req.Agent+".rego")
	if overRideRegoName != "" {
		regoPolicy = filepath.Join(dir, overRideRegoName+".rego")
	}

	r.log.Debugf("Looking up rego policy in %s", regoPolicy)
	if util.FileExist(regoPolicy) {
		r.log.Debugf("Using policy file: %s", regoPolicy)
		return regoPolicy, nil
	}

	defaultPolicy := filepath.Join(dir, "default.rego")
	if util.FileExist(defaultPolicy) {
		r.log.Debugf("Using policy file: %s", defaultPolicy)
		return defaultPolicy, nil
	}
	return "", fmt.Errorf("no policy %s found for %s in %s", defaultPolicy, r.req.Agent, dir)

}

func (r *regoPolicy) regoInputs() map[string]any {
	facts := map[string]any{}

	sif := r.si.Facts()
	err := json.Unmarshal(sif, &facts)
	if err != nil {
		r.log.Errorf("could not marshal facts for rego policy: %v", err)
	}

	data := make(map[string]any)
	err = json.Unmarshal(r.req.Data, &data)
	if err != nil {
		r.log.Errorf("could not marshal data from request: %v", err)
	}

	return map[string]any{
		"agent":          r.req.Agent,
		"action":         r.req.Action,
		"callerid":       r.req.CallerID,
		"collective":     r.req.Collective,
		"data":           data,
		"ttl":            r.req.TTL,
		"time":           r.req.Time,
		"facts":          facts,
		"classes":        r.si.Classes(),
		"agents":         r.si.KnownAgents(),
		"provision_mode": r.fw.ProvisionMode(),
	}
}

func (r *regoPolicy) enableTracing() bool {
	tracing, err := util.StrToBool(r.cfg.Option("plugin.regopolicy.tracing", "n"))
	if err != nil {
		return false
	}

	return tracing
}
