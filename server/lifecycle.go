package server

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/choria-io/go-choria/lifecycle"
)

// SetComponent sets the component name this server will report in its
// lifecycle events. "server" and "provision_mode_server" are the defaults
func (srv *Instance) SetComponent(c string) {
	srv.lifecycleComponent = c
}

// PublishEvent publishes a lifecycle event to the network
func (srv *Instance) PublishEvent(e lifecycle.Event) error {
	if srv.cfg.Choria.LegacyLifeCycleFormat {
		e.SetFormat(lifecycle.ChoriaFormat)
	} else {
		e.SetFormat(lifecycle.CloudEventV1Format)
	}

	return lifecycle.PublishEvent(e, srv.connector)
}

func (srv *Instance) eventComponent() string {
	if srv.lifecycleComponent != "" {
		return srv.lifecycleComponent
	}

	if srv.fw.ProvisionMode() {
		return "provision_mode_server"
	}

	return "server"
}

func (srv *Instance) publichShutdownEvent() {
	event, err := lifecycle.New(lifecycle.Shutdown, lifecycle.Identity(srv.cfg.Identity), lifecycle.Component(srv.eventComponent()))
	if err != nil {
		srv.log.Errorf("Could not create new shutdown event: %s", err)
		return
	}

	err = srv.PublishEvent(event)
	if err != nil {
		srv.log.Errorf("Could not publish shutdown event: %s", err)
	}
}

func (srv *Instance) publishStartupEvent() {
	event, err := lifecycle.New(lifecycle.Startup, lifecycle.Identity(srv.cfg.Identity), lifecycle.Version(srv.fw.BuildInfo().Version()), lifecycle.Component(srv.eventComponent()))
	if err != nil {
		srv.log.Errorf("Could not create new startup event: %s", err)
		return
	}

	err = srv.PublishEvent(event)
	if err != nil {
		srv.log.Errorf("Could not publish startup event: %s", err)
	}
}

func (srv *Instance) publishAliveEvents(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	delay := time.Duration(rand.Intn(60)) * time.Minute
	event, err := lifecycle.New(lifecycle.Alive, lifecycle.Identity(srv.cfg.Identity), lifecycle.Version(srv.fw.BuildInfo().Version()), lifecycle.Component(srv.eventComponent()))
	if err != nil {
		srv.log.Errorf("Could not create new alive event: %s", err)
		return
	}

	srv.log.Debugf("Sleeping %v until first alive event", delay)

	select {
	case <-time.NewTimer(delay).C:
	case <-ctx.Done():
		return
	}

	f := func() {
		srv.log.Debug("Publishing alive event")
		err = srv.PublishEvent(event)
		if err != nil {
			srv.log.Errorf("Could not publish startup event: %s", err)
		}
	}

	ticker := time.NewTicker(60 * time.Minute)

	f()

	for {
		select {
		case <-ticker.C:
			f()

		case <-ctx.Done():
			return
		}
	}
}
