package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/choria-io/go-srvcache"
)

func (srv *Instance) initialConnect(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		return fmt.Errorf("Existing on shut down")
	}

	brokers := func() (srvcache.Servers, error) {
		tempsrv, err := srv.brokerUrls(ctx)
		if err != nil {
			return nil, fmt.Errorf("Could not find Choria Network Brokers: %s", err)
		}

		list := tempsrv.Strings()

		srv.log.Infof("Choria Network Brokers: %#v", strings.Join(list, ", "))

		return tempsrv, nil
	}

	srv.connector, err = srv.fw.NewConnector(ctx, brokers, srv.fw.Certname(), srv.log)
	if err != nil {
		return fmt.Errorf("Could not create connector: %s", err)
	}

	return nil
}

func (srv *Instance) brokerUrls(ctx context.Context) (servers srvcache.Servers, err error) {
	if srv.fw.ProvisionMode() {
		servers, err = srv.fw.ProvisioningServers(ctx)
		if err != nil {
			srv.log.Errorf("Could not determine provisioning broker urls while provisioning is configured: %s", err)
		}

		// provisioning is like a flat network no federation
		// so this will disable federation when provisioning
		// and after provisioning the reload will restore
		// the configured federation setup and all will
		// continue as normal with federation and all
		if servers.Count() > 0 {
			srv.mu.Lock()
			if !srv.provisioning {
				srv.log.Infof("Entering provision mode with servers %v", servers)
				srv.provisioning = true
			}
			srv.mu.Unlock()

			return servers, nil
		}
	}

	servers, err = srv.fw.MiddlewareServers()

	return servers, err
}

func (srv *Instance) subscribeNode(ctx context.Context) error {
	var err error

	for _, collective := range srv.cfg.Collectives {
		target := srv.connector.NodeDirectedTarget(collective, srv.cfg.Identity)

		srv.log.Infof("Subscribing node %s to %s", srv.cfg.Identity, target)

		err = srv.connector.QueueSubscribe(ctx, fmt.Sprintf("node.%s", collective), target, "", srv.requests)
		if err != nil {
			return fmt.Errorf("Could not subscribe to node directed targets: %s", err)
		}
	}

	return nil
}
