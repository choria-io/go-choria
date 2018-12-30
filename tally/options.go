package tally

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// options are options that configure the tool
type options struct {
	Component  string
	Debug      bool
	Log        *logrus.Entry
	Connector  Connector
	StatPrefix string
}

// Option configures Options
type Option func(*options)

// Validate validates options meet minimal requirements, also assigns defaults
// for optional settings
func (o *options) Validate() error {
	if o.Component == "" {
		return fmt.Errorf("needs a component")
	}

	if o.Component == "*" || o.Component == ">" {
		return fmt.Errorf("invalid component %s", o.Component)
	}

	if o.Connector == nil {
		return fmt.Errorf("needs a connector")
	}

	if o.StatPrefix == "" {
		o.StatPrefix = fmt.Sprintf("lifecycle_tally")
	}

	if o.Log == nil {
		o.Log = logrus.NewEntry(logrus.New())
	}

	return nil
}

// Component sets the component to tally
func Component(c string) Option {
	return func(o *options) {
		o.Component = c
	}
}

// Debug enable debug logging
func Debug() Option {
	return func(o *options) {
		o.Debug = true
	}
}

// Logger is the logger to use
func Logger(l *logrus.Entry) Option {
	return func(o *options) {
		o.Log = l
	}
}

// Connection is the middleware to receive events on
func Connection(c Connector) Option {
	return func(o *options) {
		o.Connector = c
	}
}

// StatsPrefix is the space to create stat entries in
func StatsPrefix(p string) Option {
	return func(o *options) {
		o.StatPrefix = p
	}
}
