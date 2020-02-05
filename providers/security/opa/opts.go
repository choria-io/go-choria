package opa

import (
	"github.com/open-policy-agent/opa/rego"
	"github.com/sirupsen/logrus"
)

// Options are configuration options for the opa system
type Options struct {
	policyFile string
	policyCode []byte
	logger     *logrus.Entry
	trace      bool
	functions  []func(*rego.Rego)
}

// Option configures the opa system
type Option func(*Options) error

// File sets the file to read the policy from, mutually exclusive with Policy()
func File(f string) Option {
	return func(o *Options) error {
		o.policyFile = f
		return nil
	}
}

// Policy sets the contents of the policy to evaluate, mutually exclusive with File()
func Policy(p []byte) Option {
	return func(o *Options) error {
		o.policyCode = p
		return nil
	}
}

// Logger sets the logger to use
func Logger(log *logrus.Entry) Option {
	return func(o *Options) error {
		o.logger = log
		return nil
	}
}

// Trace enables tracing of rego evaluation
func Trace() Option {
	return func(o *Options) error {
		o.trace = true
		return nil
	}
}

// Function adds functions to the rego ast
func Function(fs ...func(*rego.Rego)) Option {
	return func(o *Options) error {
		o.functions = append(o.functions, fs...)
		return nil
	}
}
