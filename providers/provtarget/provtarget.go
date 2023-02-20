// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package provtarget

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/provtarget/builddefaults"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/srvcache"
)

// TargetResolver is capable of resolving the target brokers for provisioning into list of strings in the format host:port
type TargetResolver interface {
	// Name the display name that will be shown in places like `choria buildinfo`
	Name() string

	// Targets will be called to determine the provisioning destination
	Targets(context.Context, *logrus.Entry) []string

	// Configure will be called during server configuration and can be used to configure the target or adjust build settings or configuration
	// this will always be called even when not in provisioning mode, one can use this to programmatically set a provisioner token for example
	//
	// The intention of this function is that all the settings needed by provisioning (all the things in build) should be set during configure
	// stage.  Later when Targets() are called the intention is that either the configured targets are returned verbatim or if for example the
	// plugin queries something like SRV records those queries are done there.
	//
	// Today Configure() is expected to set the JWT file using bi.SetProvisionJWTFile() and that the file should exist before probisioning will
	// happen, this will be revisited in future. See the shouldProvision() function in server_run.go for current logic that would trigger a
	// server into provisioning.
	Configure(context.Context, *config.Config, *logrus.Entry)
}

var mu = &sync.Mutex{}
var resolver = TargetResolver(builddefaults.Provider())

// RegisterTargetResolver registers a custom target resolver, else the default will be used
func RegisterTargetResolver(r TargetResolver) error {
	mu.Lock()
	defer mu.Unlock()

	resolver = r

	return nil
}

// Configure allows the resolver to adjust configuration
func Configure(ctx context.Context, cfg *config.Config, log *logrus.Entry) {
	mu.Lock()
	defer mu.Unlock()

	if resolver == nil {
		return
	}

	resolver.Configure(ctx, cfg, log)
}

// Targets is a list of brokers to connect to
func Targets(ctx context.Context, log *logrus.Entry) (srvcache.Servers, error) {
	mu.Lock()
	defer mu.Unlock()

	if resolver == nil {
		return srvcache.NewServers(), fmt.Errorf("no Provisioning Target Resolver registered")
	}

	s := resolver.Targets(ctx, log)

	if len(s) == 0 {
		return srvcache.NewServers(), fmt.Errorf("provisioning target plugin %s returned no servers", Name())
	}

	servers, err := srvcache.StringHostsToServers(s, "nats")
	if err != nil {
		return srvcache.NewServers(), fmt.Errorf("could not determine provisioning servers using %s provisioning target plugin: %s", Name(), err)
	}

	if servers.Count() == 0 {
		return srvcache.NewServers(), fmt.Errorf("provisioning broker urls from the %s plugin were not in the valid format, 0 server:port combinations were found in %v", Name(), s)
	}

	return servers, nil
}

// Name is the name of the plugin used
func Name() string {
	if resolver == nil {
		return "Unknown"
	}

	return resolver.Name()
}
