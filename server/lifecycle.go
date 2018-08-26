package server

import (
	"github.com/choria-io/go-choria/build"
	lifecycle "github.com/choria-io/go-lifecycle"
)

func (srv *Instance) publishStartupEvent() {
	opts := []lifecycle.Option{
		lifecycle.Identity(srv.cfg.Identity),
		lifecycle.Version(build.Version),
	}

	if srv.fw.ProvisionMode() {
		opts = append(opts, lifecycle.Component("provision_mode_server"))
	} else {
		opts = append(opts, lifecycle.Component("server"))
	}

	event, err := lifecycle.New(lifecycle.Startup, opts...)
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
