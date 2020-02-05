package mcorpc

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-config"
	"github.com/choria-io/go-security/opa"
	"github.com/open-policy-agent/opa/ast"
	"github.com/sirupsen/logrus"
)

type regoPolicy struct {
	cfg   *config.Config
	req   *Request
	agent *Agent
	log   *logrus.Entry
}

func regoPolicyAuthorize(req *Request, agent *Agent, log *logrus.Entry) (bool, error) {
	logger := log.WithFields(logrus.Fields{
		"authorizer": "regoPolicy",
		"agent":      agent.Name(),
		"request":    req.RequestID,
	})

	authz := &regoPolicy{
		cfg:   agent.Config,
		req:   req,
		agent: agent,
		log:   logger,
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

	allowed, err := evaluator.Evaluate(context.Background(), r.regoInputs())
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

func (r *regoPolicy) lookupPolicyFile() (string, error) {
	dir := filepath.Join(filepath.Dir(r.cfg.ConfigFile), "policies", "rego")

	regoPolicy := filepath.Join(dir, r.agent.Name()+".rego")

	r.log.Debugf("Looking up rego policy in %s", regoPolicy)
	if choria.FileExist(regoPolicy) {
		r.log.Debugf("Using policy file: %s", regoPolicy)
		return regoPolicy, nil
	}

	defaultPolicy := filepath.Join(dir, "default.rego")
	if choria.FileExist(defaultPolicy) {
		r.log.Debugf("Using policy file: %s", defaultPolicy)
		return defaultPolicy, nil
	}
	return "", fmt.Errorf("no policy %s found for %s in %s", defaultPolicy, r.agent.Name(), dir)

}

func (r *regoPolicy) regoInputs() map[string]interface{} {
	facts := map[string]interface{}{}

	err := json.Unmarshal(r.agent.ServerInfoSource.Facts(), &facts)
	if err != nil {
		r.log.Errorf("could not marshal facts for rego policy: %v", err)
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(r.req.Data, &data)
	if err != nil {
		r.log.Errorf("could not marshal data from request: %v", err)
	}

	return map[string]interface{}{
		"agent":          r.req.Agent,
		"action":         r.req.Action,
		"callerid":       r.req.CallerID,
		"collective":     r.req.Collective,
		"data":           data,
		"ttl":            r.req.TTL,
		"time":           r.req.Time,
		"facts":          facts,
		"classes":        r.agent.ServerInfoSource.Classes(),
		"agents":         r.agent.ServerInfoSource.KnownAgents(),
		"provision_mode": r.agent.Choria.ProvisionMode(),
	}
}

func (r *regoPolicy) enableTracing() bool {
	tracing, err := choria.StrToBool(r.cfg.Option("plugin.regopolicy.tracing", "n"))
	if err != nil {
		return false
	}

	return tracing
}
