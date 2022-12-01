// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package mcorpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/opa"
	"github.com/choria-io/go-choria/tokens"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"
	"github.com/sirupsen/logrus"
)

type aaasvcPolicy struct {
	cfg   *config.Config
	req   *Request
	agent *Agent
	log   *logrus.Entry
}

func aaasvcPolicyAuthorize(req *Request, agent *Agent, log *logrus.Entry) (bool, error) {
	logger := log.WithFields(logrus.Fields{
		"authorizer": "aaasvc",
		"agent":      agent.Name(),
		"request":    req.RequestID,
	})

	authz := &aaasvcPolicy{
		cfg:   agent.Config,
		req:   req,
		agent: agent,
		log:   logger,
	}

	return authz.authorize()
}

func (r *aaasvcPolicy) authorize() (bool, error) {
	if r.req.CallerPublicData == "" {
		return false, fmt.Errorf("no policy received in request")
	}

	claims, err := tokens.ParseClientIDTokenUnverified(r.req.CallerPublicData)
	if err != nil {
		return false, fmt.Errorf("invalid token in request: %v", err)
	}

	if r.req.Agent == "discovery" {
		r.log.Debugf("Allowing discovery request")
		return true, nil
	}

	allowed := false
	hasAgents := len(claims.AllowedAgents) > 0
	hasOpa := claims.OPAPolicy != ""

	switch {
	case !(hasAgents || hasOpa):
		return false, fmt.Errorf("no policy received in token")
	case hasAgents && hasOpa:
		return false, fmt.Errorf("received agent list and rego policy")
	case hasAgents:
		r.log.Debugf("Processing using agent list")

		allowed, err = EvaluateAgentListPolicy(r.req.Agent, r.req.Action, claims.AllowedAgents, r.log)
	case hasOpa:
		r.log.Debugf("Processing using opa policy")

		allowed, err = EvaluateOpenPolicyAgentPolicy(r.req, claims.OPAPolicy, claims, "server", r.log)
	}

	return allowed, err
}

func EvaluateAgentListPolicy(agent string, action string, policy []string, _ *logrus.Entry) (bool, error) {
	if len(policy) == 0 {
		return false, nil
	}

	for _, allow := range policy {
		// all things are allowed
		if allow == "*" {
			return true, nil
		}

		parts := strings.Split(allow, ".")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid agent policy: %s", allow)
		}

		// it's a claim for a different agent so pass, no need to check it here
		if agent != parts[0] {
			continue
		}

		// agent matches, action is * so allow it
		if parts[1] == "*" {
			return true, nil
		}

		// agent matches, action matches, allow it
		if action == parts[1] {
			return true, nil
		}
	}

	return false, nil

}

// EvaluateOpenPolicyAgentPolicy evaluates a rego policy document, typically embedded in a JWT token, against a request.  Shared by Choria and AAA Service
func EvaluateOpenPolicyAgentPolicy(req *Request, policy string, claims *tokens.ClientIDClaims, site string, log *logrus.Entry) (allowed bool, err error) {
	if policy == "" {
		return false, fmt.Errorf("invalid policy given")
	}

	eopts := []opa.Option{
		opa.Logger(log),
		opa.Policy([]byte(policy)),
		opa.Function(opaFunctionsMap(req)...),
	}

	if log.Logger.GetLevel() == logrus.DebugLevel {
		eopts = append(eopts, opa.Trace())
	}

	evaluator, err := opa.New("io.choria.aaasvc", "data.io.choria.aaasvc.allow", eopts...)
	if err != nil {
		return false, fmt.Errorf("could not initialize opa evaluator: %v", err)
	}

	inputs, err := opaInputs(req, req.Data, site, claims)
	if err != nil {
		return false, err
	}

	allowed, err = evaluator.Evaluate(context.Background(), inputs)
	if err != nil {
		return false, err
	}

	return allowed, nil
}

func opaInputs(req *Request, data json.RawMessage, site string, claims *tokens.ClientIDClaims) (map[string]any, error) {
	dat := map[string]any{}
	err := json.Unmarshal(data, &dat)
	if err != nil {
		return nil, err
	}

	// lame deep copy/data convert thing happening here
	jclaims, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("could not JSON encode claims")
	}

	cdat := new(map[string]any)
	err = json.Unmarshal(jclaims, &cdat)
	if err != nil {
		return nil, fmt.Errorf("could not JSON encode claims")
	}

	return map[string]any{
		"agent":      req.Agent,
		"action":     req.Action,
		"data":       data,
		"sender":     req.SenderID,
		"collective": req.Collective,
		"ttl":        req.TTL,
		"time":       req.Time,
		"site":       site,
		"claims":     cdat,
	}, nil
}

func opaFunctionsMap(req *Request) []func(r *rego.Rego) {
	return []func(r *rego.Rego){
		rego.Function1(&rego.Function{Name: "requires_filter", Decl: types.NewFunction(types.Args(), types.B)}, opaFuncRequiresFilter(req)),
		rego.Function1(&rego.Function{Name: "requires_fact_filter", Decl: types.NewFunction(types.Args(types.S), types.B)}, opaFuncRequiresFactFilter(req)),
		rego.Function1(&rego.Function{Name: "requires_class_filter", Decl: types.NewFunction(types.Args(types.S), types.B)}, opaFuncRequiresClassFilter(req)),
		rego.Function1(&rego.Function{Name: "requires_identity_filter", Decl: types.NewFunction(types.Args(types.S), types.B)}, opaFuncRequiresIdentityFilter(req)),
	}
}

func opaFuncRequiresFilter(req *Request) func(_ rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
		// agent is always set, so we don't check it else it will always be true
		if len(req.Filter.ClassFilters()) > 0 || len(req.Filter.IdentityFilters()) > 0 || len(req.Filter.FactFilters()) > 0 || len(req.Filter.CompoundFilters()) > 0 {
			return ast.BooleanTerm(true), nil
		}

		return ast.BooleanTerm(false), nil
	}

}

func opaFuncRequiresIdentityFilter(req *Request) func(_ rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
		str, ok := a.Value.(ast.String)
		if !ok {
			return ast.BooleanTerm(false), fmt.Errorf("invalid identity matcher received")
		}

		want := string(str)
		for _, f := range req.Filter.IdentityFilters() {
			if f == want {
				return ast.BooleanTerm(true), nil
			}
		}

		return ast.BooleanTerm(false), nil
	}
}

func opaFuncRequiresClassFilter(req *Request) func(_ rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
		str, ok := a.Value.(ast.String)
		if !ok {
			return ast.BooleanTerm(false), fmt.Errorf("invalid class matcher received")
		}

		want := string(str)

		for _, f := range req.Filter.ClassFilters() {
			if f == want {
				return ast.BooleanTerm(true), nil
			}
		}

		return ast.BooleanTerm(false), nil
	}
}

func opaFuncRequiresFactFilter(req *Request) func(_ rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
	return func(_ rego.BuiltinContext, a *ast.Term) (*ast.Term, error) {
		str, ok := a.Value.(ast.String)
		if !ok {
			return ast.BooleanTerm(false), fmt.Errorf("invalid fact matcher received")
		}

		want, err := client.ParseFactFilterString(string(str))
		if err != nil {
			return ast.BooleanTerm(false), fmt.Errorf("invalid fact matcher received: %s", err)
		}

		for _, f := range req.Filter.Fact {
			if want.Fact == f.Fact && want.Operator == f.Operator && want.Value == f.Value {
				return ast.BooleanTerm(true), nil
			}
		}

		return ast.BooleanTerm(false), nil
	}
}
