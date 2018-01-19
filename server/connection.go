package server

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/choria"
)

func (srv *Instance) initialConnect(ctx context.Context) error {
	if ctx.Err() != nil {
		return fmt.Errorf("Existing on shut down")
	}

	tempsrv, err := srv.brokerUrls()
	if err != nil {
		return fmt.Errorf("Could not find initial NATS servers: %s", err.Error())
	}

	srv.log.Infof("Initial servers: %#v", tempsrv)

	srv.connector, err = srv.fw.NewConnector(ctx, srv.brokerUrls, srv.fw.Certname(), srv.log)
	if err != nil {
		return fmt.Errorf("Could not create connector: %s", err.Error())
	}

	return nil
}

func (srv *Instance) brokerUrls() ([]choria.Server, error) {
	servers := []choria.Server{}
	var err error

	if srv.fw.ProvisionMode() {
		servers, err = srv.fw.ProvisioningServers()
		if err != nil {
			srv.log.Errorf("Could not determine provisioning broker urls while provisioning is configured: %s", err)
		}

		// provisioning is like a flat network no federation
		// so this will disable federation when provisioning
		// and after provisioning the reload will restore
		// the configured federation setup and all will
		// continue as normal with federation and all
		if len(servers) > 0 {
			srv.mu.Lock()
			if !srv.provisioning {
				srv.log.Infof("Entering provision mode with servers %v and federation disabled", servers)
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
			return fmt.Errorf("Could not subscribe to node directed targets: %s", err.Error())
		}
	}

	return nil
}
