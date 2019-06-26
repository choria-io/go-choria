package provtarget

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/provtarget/builddefaults"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-srvcache"
)

// TargetResolver is capable of resolving the target brokers for provisioning into list of strings in the format host:port
type TargetResolver interface {
	// Name the display name that will be shown in places like `choria buildinfo`
	Name() string

	// Targets will be called to determine the provisioning destination
	Targets(context.Context, *logrus.Entry) []string
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
		return srvcache.NewServers(), fmt.Errorf("provisioning broker urls from the %s plugin were not in the valid format, 0 server:port combinations were foundin %v", Name(), s)
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
