package server

import (
	"github.com/choria-io/go-choria/build"
	lifecycle "github.com/choria-io/go-lifecycle"
)

// SetComponent sets the component name this server will report in its
// lifecycle events. "server" and "provision_mode_server" are the defaults
func (srv *Instance) SetComponent(c string) {
	srv.lifecycleComponent = c
}

// PublishEvent publishes a lifecycle event to the network
func (srv *Instance) PublishEvent(e lifecycle.Event) error {
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
	event, err := lifecycle.New(lifecycle.Startup, lifecycle.Identity(srv.cfg.Identity), lifecycle.Version(build.Version), lifecycle.Component(srv.eventComponent()))
	if err != nil {
		srv.log.Errorf("Could not create new startup event: %s", err)
		return
	}

	err = srv.PublishEvent(event)
	if err != nil {
		srv.log.Errorf("Could not publish startup event: %s", err)
	}
}
