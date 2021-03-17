package network

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/scout"
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

func (s *Server) configureSystemStreams(ctx context.Context) error {
	if !s.opts.JetStream {
		return nil
	}

	var opts []nats.Option

	if s.IsTLS() {
		s.log.Info("Connecting to Choria Stream using TLS")
		opts = append(opts, nats.Secure(s.opts.TLSConfig))
	} else {
		s.log.Info("Configuring Choria System Streams without TLS")
	}

	var nc *nats.Conn
	var err error

	err = backoff.TwentySec.For(ctx, func(try int) error {
		nc, err = nats.Connect(s.opts.ClientAdvertise, opts...)
		if err != nil {
			s.log.Warnf("Could not connect to broker %s to configure System Streams: %s", s.opts.ClientAdvertise, err)
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	defer nc.Close()

	mgr, err := jsm.New(nc)
	if err != nil {
		return err
	}

	err = s.createOrUpdateStream("CHORIA_EVENTS", []string{"choria.lifecycle.>"}, s.config.Choria.NetworkEventStoreDuration, s.config.Choria.NetworkEventStoreReplicas, mgr)
	if err != nil {
		return err
	}

	err = s.createOrUpdateStream("CHORIA_MACHINE", []string{"choria.machine.>"}, s.config.Choria.NetworkEventStoreDuration, s.config.Choria.NetworkEventStoreReplicas, mgr)
	if err != nil {
		return err
	}

	err = s.createOrUpdateStream("CHORIA_STREAM_ADVISORIES", []string{"$JS.EVENT.ADVISORY.>"}, s.config.Choria.NetworkEventStoreDuration, s.config.Choria.NetworkEventStoreReplicas, mgr)
	if err != nil {
		return err
	}

	err = scout.ConfigureStreams(nc, s.log.WithField("component", "scout"))
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) createOrUpdateStream(name string, subjects []string, maxAge time.Duration, replicas int, mgr *jsm.Manager) error {
	if int(maxAge) <= 0 {
		return nil
	}

	str, err := mgr.NewStream(name, jsm.FileStorage(), jsm.Subjects(subjects...), jsm.MaxAge(s.config.Choria.NetworkEventStoreDuration), jsm.Replicas(replicas))
	if err != nil {
		return fmt.Errorf("could not load or create %s: %s", name, err)
	}

	cfg := str.Configuration()
	if cfg.MaxAge != maxAge {
		cfg.MaxAge = maxAge
		err = str.UpdateConfiguration(cfg)
		if err != nil {
			return fmt.Errorf("could not update retention period for %s Stream: %s", name, err)
		}
	}

	return nil
}
