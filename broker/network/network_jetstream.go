package network

import (
	"github.com/nats-io/jsm.go"
)

func (s *Server) setupStreaming() error {
	if s.config.Choria.NetworkStreamStore == "" {
		return nil
	}

	s.log.Infof("Configuring Choria Stream Processing in %v", s.config.Choria.NetworkStreamStore)

	s.opts.JetStream = true
	s.opts.StoreDir = s.config.Choria.NetworkStreamStore

	return nil
}

func (s *Server) configureSystemStreams() error {
	if int(s.config.Choria.NetworkEventStoreDuration) > 0 {
		_, err := jsm.LoadOrNewStream("CHORIA_EVENTS", jsm.FileStorage(), jsm.Subjects("choria.lifecycle.>"), jsm.MaxAge(s.config.Choria.NetworkEventStoreDuration))
		if err != nil {
			s.log.Errorf("Could not create CHORIA_EVENTS Stream: %s", err)
		}
	}

	if int(s.config.Choria.NetworkMachineStoreDuration) > 0 {
		_, err := jsm.LoadOrNewStream("CHORIA_MACHINE", jsm.FileStorage(), jsm.Subjects("choria.lifecycle.>"), jsm.MaxAge(s.config.Choria.NetworkMachineStoreDuration))
		if err != nil {
			s.log.Errorf("Could not create CHORIA_MACHINE Stream: %s", err)
		}
	}

	return nil
}
