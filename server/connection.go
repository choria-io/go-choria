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

	servers := func() ([]choria.Server, error) {
		return srv.c.MiddlewareServers()
	}

	_, err := servers()
	if err != nil {
		return fmt.Errorf("Could not find initial NATS servers: %s", err.Error())
	}

	srv.connector, err = srv.c.NewConnector(ctx, servers, srv.c.Certname(), srv.log)
	if err != nil {
		return fmt.Errorf("Could not create connector: %s", err.Error())
	}

	return nil
}
