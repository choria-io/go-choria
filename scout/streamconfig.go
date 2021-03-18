package scout

import (
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

func ConfigureStreams(nc *nats.Conn, log *logrus.Entry) error {
	// TODO: disabled while redesigning scout standalone

	// mgr, err := jsm.New(nc)
	// if err != nil {
	// 	return err
	// }

	// _, err = mgr.LoadOrNewStream("SCOUT_TAGS", jsm.FileStorage(), jsm.MaxMessages(10000), jsm.Subjects("scout.tags.>"))
	// if err != nil {
	// 	return fmt.Errorf("could not create SCOUT_TAGS stream: %s", err)
	// }

	// cfg, err := jsm.NewStreamConfiguration(jsm.DefaultStream, jsm.FileStorage(), jsm.Subjects("scout.check.*"), jsm.MaxMessages(10))
	// if err != nil {
	// 	return fmt.Errorf("could not create SCOUT_CHECKS template configuration")
	// }

	// _, err = mgr.LoadOrNewStreamTemplate("SCOUT_CHECKS", 1000, *cfg)
	// if err != nil {
	// 	return fmt.Errorf("could not create SCOUT_CHECKS stream template: %s", err)
	// }

	// _, err = mgr.LoadOrNewStream("SCOUT_OVERRIDES", jsm.FileStorage(), jsm.Subjects("scout.overrides.>"))
	// if err != nil {
	// 	return fmt.Errorf("could not create SCOUT_OVERRIDES: %s", err)
	// }

	return nil
}
