package opa

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/topdown"
	"github.com/sirupsen/logrus"
)

type Evaluator struct {
	module string
	query  string
	log    *logrus.Entry
	opts   *Options
	pq     *rego.PreparedEvalQuery
}

func New(moduleName string, query string, opts ...Option) (eval *Evaluator, err error) {
	eval = &Evaluator{
		module: moduleName,
		query:  query,
		opts:   &Options{},
	}

	for _, opt := range opts {
		err = opt(eval.opts)
		if err != nil {
			return nil, err
		}
	}

	if eval.opts.logger != nil {
		eval.log = eval.opts.logger.WithFields(logrus.Fields{"module": moduleName})
	} else {
		eval.log = logrus.NewEntry(logrus.New()).WithFields(logrus.Fields{"module": moduleName})
	}

	err = eval.prepare()
	if err != nil {
		return nil, err
	}

	return eval, nil
}

func (e *Evaluator) prepare() error {
	policy, err := e.policy()
	if err != nil {
		return err
	}

	opts := []func(r *rego.Rego){
		rego.Query(e.query),
		rego.Module(e.module, string(policy)),
	}

	if len(e.opts.functions) > 0 {
		opts = append(opts, e.opts.functions...)
	}

	pq, err := rego.New(opts...).PrepareForEval(context.Background())
	if err != nil {
		return err
	}

	e.pq = &pq

	return nil
}

func (e *Evaluator) Evaluate(ctx context.Context, inputs interface{}) (pass bool, err error) {
	var buf *topdown.BufferTracer

	opts := []rego.EvalOption{rego.EvalInput(inputs)}

	if e.opts.trace {
		buf = topdown.NewBufferTracer()
		opts = append(opts, rego.EvalTracer(buf))
	}

	rs, err := e.pq.Eval(ctx, opts...)
	if e.opts.trace {
		topdown.PrettyTrace(e.log.Writer(), *buf)
	}
	if err != nil {
		return false, fmt.Errorf("could not evaluate rego policy %s: %s", e.module, err)
	}

	if len(rs) != 1 {
		return false, fmt.Errorf("invalid result from rego policy %s: expected 1 received %d", e.module, len(rs))
	}

	pass, ok := rs[0].Expressions[0].Value.(bool)
	if !ok {
		return false, fmt.Errorf("did not receive a boolean for 'allow' from rego evaluation of %s", e.module)
	}

	return pass, nil
}

func (e *Evaluator) policy() (p []byte, err error) {
	if len(e.opts.policyCode) > 0 {
		return e.opts.policyCode, nil
	}

	if e.opts.policyFile == "" {
		return nil, fmt.Errorf("neither Code nor File has been set, no policy to evaluate")
	}

	p, err = ioutil.ReadFile(e.opts.policyFile)
	if err != nil {
		return nil, err
	}

	return p, nil
}
