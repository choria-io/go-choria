package testutil

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-config"
)

type ChoriaServer struct {
	Instance *server.Instance
	fw       *choria.Framework
	broker   *Broker
	cfg      *config.Config
	wg       *sync.WaitGroup
	ctx      context.Context
	cancel   func()
}

func (c *ChoriaServer) Start() (err error) {
	if c.Instance != nil {
		return fmt.Errorf("instance already exist, cannot start again")
	}

	if c.broker.ClientURL() == "" {
		return fmt.Errorf("client url for broker is empty, cannot start")
	}

	c.cfg.Choria.MiddlewareHosts = []string{c.broker.ClientURL()}
	c.cfg.DisableTLS = true

	c.fw, err = choria.NewWithConfig(c.cfg)
	if err != nil {
		return err
	}

	c.Instance, err = server.NewInstance(c.fw)
	if err != nil {
		return err
	}

	c.wg.Add(1)
	return c.Instance.Run(c.ctx, c.wg)
}

func (c *ChoriaServer) Stop() {
	c.cancel()
	c.wg.Wait()
	c.Instance = nil
}
