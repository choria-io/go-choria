package builddefaults

import (
	"context"
	"strings"

	"github.com/choria-io/go-choria/build"
	"github.com/sirupsen/logrus"
)

// Provider creates an instance of the provider
func Provider() *Resolver {
	return &Resolver{}
}

// Resolver resolve names against the compile time build properties
type Resolver struct{}

// Name is te name of the resolver
func (b *Resolver) Name() string {
	return "Default"
}

// Targets are the build time configured provisioners
func (b *Resolver) Targets(ctx context.Context, log *logrus.Entry) []string {
	if build.ProvisionBrokerURLs != "" {
		return strings.Split(build.ProvisionBrokerURLs, ",")
	}

	return []string{}
}
