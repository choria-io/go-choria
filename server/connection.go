package server

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/choria"
)

func (self *Instance) initialConnect(ctx context.Context) error {
	if ctx.Err() != nil {
		return fmt.Errorf("Existing on shut down")
	}

	servers := func() ([]choria.Server, error) {
		return self.c.MiddlewareServers()
	}

	_, err := servers()
	if err != nil {
		return fmt.Errorf("Could not find initial NATS servers: %s", err.Error())
	}

	self.connector, err = self.c.NewConnector(ctx, servers, self.c.Certname(), self.log)
	if err != nil {
		return fmt.Errorf("Could not create connector: %s", err.Error())
	}

	return nil
}
