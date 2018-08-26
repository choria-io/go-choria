package server

import (
	"github.com/choria-io/go-choria/build"
	lifecycle "github.com/choria-io/go-lifecycle"
)

func (srv *Instance) eventComponent() string {
	if srv.fw.ProvisionMode() {
		return ("provision_mode_server")
	}

	return ("server")
}

func (srv *Instance) publichShutdownEvent() {
	event, err := lifecycle.New(lifecycle.Shutdown, lifecycle.Identity(srv.cfg.Identity), lifecycle.Component(srv.eventComponent()))
	if err != nil {
		srv.log.Errorf("Could not create new shutdown event: %s", err)
		return
	}

	srv.log.Debugf("Publishing shutdown event %#v", event)

	err = lifecycle.PublishEvent(event, srv.connector)
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

	srv.log.Debugf("Publishing startup event %#v", event)

	err = lifecycle.PublishEvent(event, srv.connector)
	if err != nil {
		srv.log.Errorf("Could not publish startup event: %s", err)
	}
}
