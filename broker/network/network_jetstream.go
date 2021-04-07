package network

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/scout"
)

func (s *Server) setupStreaming() error {
	if s.config.Choria.NetworkStreamStore == "" {
		return nil
	}

	if s.gnatsd.SystemAccount() == nil {
		return fmt.Errorf("system Account is required for Choria Streams")
	}

	s.log.Infof("Configuring Choria Stream Processing in %v", s.config.Choria.NetworkStreamStore)

	s.gnatsd.EnableJetStream(&server.JetStreamConfig{StoreDir: s.config.Choria.NetworkStreamStore})

	err := s.choriaAccount.EnableJetStream(&server.JetStreamAccountLimits{})
	if err != nil {
		s.log.Errorf("Could not enable Choria Streams for the %s account: %s", s.choriaAccount.Name, err)
	}

	if !s.choriaAccount.JetStreamEnabled() {
		s.log.Errorf("Choria Streams enabled for account %q but it's not reporting as enabled", s.choriaAccount.Name)
	}

	return nil
}

func (s *Server) configureSystemStreams(ctx context.Context) error {
	if s.config.Choria.NetworkStreamStore == "" {
		return nil
	}

	var opts []nats.Option

	if s.IsTLS() {
		s.log.Info("Configuring Choria System Streams with TLS")
		tlsc, err := s.choria.ClientTLSConfig()
		if err != nil {
			return err
		}
		opts = append(opts, nats.Secure(tlsc))
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

	err = s.createOrUpdateStream("CHORIA_MACHINE", []string{"choria.machine.>"}, s.config.Choria.NetworkMachineStoreDuration, s.config.Choria.NetworkMachineStoreReplicas, mgr)
	if err != nil {
		return err
	}

	err = s.createOrUpdateStream("CHORIA_STREAM_ADVISORIES", []string{"$JS.EVENT.ADVISORY.>"}, s.config.Choria.NetworkStreamAdvisoryDuration, s.config.Choria.NetworkStreamAdvisoryReplicas, mgr)
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

	str, err := mgr.LoadOrNewStream(name, jsm.FileStorage(), jsm.Subjects(subjects...), jsm.MaxAge(maxAge), jsm.Replicas(replicas))
	if err != nil {
		return fmt.Errorf("could not load or create %s: %s", name, err)
	}

	cfg := str.Configuration()
	if cfg.MaxAge != maxAge {
		s.log.Infof("Updating %s retention from %s to %s", str.Name(), cfg.MaxAge, maxAge)
		cfg.MaxAge = maxAge
		err = str.UpdateConfiguration(cfg)
		if err != nil {
			return fmt.Errorf("could not update retention period for %s Stream: %s", name, err)
		}
	}

	s.log.Infof("Configured stream %q with %d replicas and %s retention", name, replicas, maxAge)

	return nil
}
