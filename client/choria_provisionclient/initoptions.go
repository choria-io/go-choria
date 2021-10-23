// generated code; DO NOT EDIT

package choria_provisionclient

import (
	"time"

	"github.com/sirupsen/logrus"
)

type initOptions struct {
	cfgFile  string
	logger   *logrus.Entry
	ns       NodeSource
	dt       time.Duration
	progress bool
}

// InitializationOption is an optional setting used to initialize the client
type InitializationOption func(opts *initOptions)

// Logger sets the logger to use else one is made via the Choria framework
func Logger(l *logrus.Entry) InitializationOption {
	return func(o *initOptions) {
		o.logger = l
	}
}

// Discovery sets the NodeSource to use when finding nodes to manage
func Discovery(ns NodeSource) InitializationOption {
	return func(o *initOptions) {
		o.ns = ns
	}
}

// Progress enables displaying a progress bar
func Progress() InitializationOption {
	return func(o *initOptions) {
		o.progress = true
	}
}

// DiscoveryMethod accepts a discovery method name as supplied from the CLI and configures the correct NodeSource
// reverts to broadcast method if an unsupported method is supplied, custom node sources can be set using Discovery()
func DiscoveryMethod(m string) InitializationOption {
	return func(o *initOptions) {
		switch m {
		case "choria", "puppetdb", "pdb":
			o.ns = &PuppetDBNS{}
		default:
			o.ns = &BroadcastNS{}
		}
	}
}

// DiscoveryTimeout sets a timeout for discovery for those methods that support it
func DiscoveryTimeout(t time.Duration) InitializationOption {
	return func(o *initOptions) {
		o.dt = t
	}
}
