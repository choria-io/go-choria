package federation

import (
	"context"
	"strings"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

func NewChoriaNatsEgest(workers int, mode int, capacity int, broker *FederationBroker, logger *log.Entry) (*pooledWorker, error) {
	worker, err := PooledWorkerFactory("choria_nats_egest", workers, mode, capacity, broker, logger, func(ctx context.Context, self *pooledWorker, i int, logger *log.Entry) {
		defer self.wg.Done()

		var nc choria.Connector
		var err error

		nc, err = self.connection.NewConnector(self.servers, self.Name(), logger)
		if err != nil {
			logger.Errorf("Could not start NATS connection for worker %d: %s", i, err.Error())
			return
		}

		for {
			var cm chainmessage

			select {
			case cm = <-self.in:
			case <-ctx.Done():
				logger.Infof("Worker routine %s exiting", self.Name())
				return
			}

			if len(cm.Targets) == 0 {
				logger.Errorf("Received message '%s' with no targets, discarding: %#v", cm.RequestID, cm)
				continue
			}

			logger.Debugf("Publishing message '%s' to %d target(s)", cm.RequestID, len(cm.Targets))

			cm.Seen = append(cm.Seen, self.Name())
			cm.Seen = append(cm.Seen, nc.ConnectedServer())

			if len(cm.Seen) >= 3 {
				cm.Message.RecordNetworkHop(cm.Seen[0], strings.Join(cm.Seen[1:len(cm.Seen)-1], ", "), cm.Seen[len(cm.Seen)-1])
			}

			j, err := cm.Message.JSON()
			if err != nil {
				logger.Errorf("Could not JSON encode message '%s': %s", cm.RequestID, err.Error())
				continue
			}

			for _, target := range cm.Targets {
				if err = nc.PublishRaw(target, []byte(j)); err != nil {
					logger.Errorf("Could not publish message '%s' to '%s': %s", cm.RequestID, target, err.Error())
				}
			}
		}
	})

	return worker, err
}
