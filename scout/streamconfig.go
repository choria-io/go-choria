package scout

import (
	"fmt"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

func ConfigureStreams(nc *nats.Conn, log *logrus.Entry) error {
	conn := []jsm.RequestOption{
		jsm.WithConnection(nc),
	}

	_, err := jsm.LoadOrNewStream("SCOUT_TAGS", jsm.FileStorage(), jsm.MaxMessages(10000), jsm.Subjects("scout.tags.>"), jsm.StreamConnection(conn...))
	if err != nil {
		return fmt.Errorf("could not create SCOUT_TAGS stream: %s", err)
	}

	cfg, err := jsm.NewStreamConfiguration(jsm.DefaultStream, jsm.FileStorage(), jsm.Subjects("scout.check.*"), jsm.MaxMessages(10))
	if err != nil {
		return fmt.Errorf("could not create SCOUT_CHECKS template configuration")
	}

	_, err = jsm.LoadOrNewStreamTemplate("SCOUT_CHECKS", 1000, cfg.StreamConfig, jsm.StreamConnection(conn...))
	if err != nil {
		return fmt.Errorf("could not create SCOUT_CHECKS stream template: %s", err)
	}

	_, err = jsm.LoadOrNewStream("SCOUT_OVERRIDES", jsm.FileStorage(), jsm.Subjects("scout.overrides.>"), jsm.StreamConnection(conn...))
	if err != nil {
		return fmt.Errorf("could not create SCOUT_OVERRIDES: %s", err)
	}

	return nil
}
