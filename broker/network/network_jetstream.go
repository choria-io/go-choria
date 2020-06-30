package network

import (
	"fmt"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
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
	var opts []nats.Option

	if s.IsTLS() {
		opts = append(opts, nats.Secure(s.opts.TLSConfig))
	}

	nc, err := nats.Connect(s.gnatsd.ClientURL(), opts...)
	if err != nil {
		s.log.Errorf("could not connect to configure Choria Streams: %s", err)
		return nil
	}
	defer nc.Close()

	if int(s.config.Choria.NetworkEventStoreDuration) > 0 {
		known, err := jsm.IsKnownStream("CHORIA_EVENTS", jsm.WithConnection(nc))
		if err != nil {
			return fmt.Errorf("could not determine if Stream CHORIA_EVENTS exist: %s", err)
		}

		if !known {
			str, err := jsm.NewStream("CHORIA_EVENTS", jsm.FileStorage(), jsm.Subjects("choria.lifecycle.>"), jsm.MaxAge(s.config.Choria.NetworkEventStoreDuration), jsm.StreamConnection(jsm.WithConnection(nc)))
			if err != nil {
				return fmt.Errorf("could not create CHORIA_EVENTS Stream: %s", err)
			}
			s.log.Infof("Created stream %s with retention %v", str.Name(), str.MaxAge())
		}
	}

	if int(s.config.Choria.NetworkMachineStoreDuration) > 0 {
		known, err := jsm.IsKnownStream("CHORIA_MACHINE", jsm.WithConnection(nc))
		if err != nil {
			return fmt.Errorf("could not determine if Stream CHORIA_EVENTS exist: %s", err)
		}

		if !known {
			str, err := jsm.NewStream("CHORIA_MACHINE", jsm.FileStorage(), jsm.Subjects("choria.machine.>"), jsm.MaxAge(s.config.Choria.NetworkEventStoreDuration), jsm.StreamConnection(jsm.WithConnection(nc)))
			if err != nil {
				return fmt.Errorf("could not create CHORIA_MACHINE Stream: %s", err)
			}
			s.log.Infof("Created stream %s with retention %v", str.Name(), str.MaxAge())
		}
	}

	return nil
}
