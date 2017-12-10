package federation

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

func NewChoriaNatsIngest(workers int, mode int, capacity int, broker *FederationBroker, logger *log.Entry) (*pooledWorker, error) {
	worker, err := PooledWorkerFactory("choria_nats_ingest", workers, mode, capacity, broker, logger, func(ctx context.Context, self *pooledWorker, i int, logger *log.Entry) {
		defer self.wg.Done()

		nc, err := self.connection.NewConnector(ctx, self.servers, self.Name(), logger)
		if err != nil {
			logger.Errorf("Could not start NATS connection for worker %d: %s", i, err.Error())
			return
		}

		var grp, subj string

		switch self.mode {
		case Federation:
			subj = fmt.Sprintf("choria.federation.%s.federation", self.broker.Name)
			grp = fmt.Sprintf("%s_federation", self.broker.Name)
		case Collective:
			subj = fmt.Sprintf("choria.federation.%s.collective", self.broker.Name)
			grp = fmt.Sprintf("%s_collective", self.broker.Name)
		}

		natsch, err := nc.ChanQueueSubscribe("ingest", subj, grp, 64)
		if err != nil {
			logger.Errorf("Could not subscribe to %s: %s", subj, err.Error())
			return
		}

		for {
			var msg *choria.ConnectorMessage

			select {
			case msg = <-natsch:
			case <-ctx.Done():
				logger.Infof("Worker routine %s exiting", self.Name())
				return
			}

			message, err := self.choria.NewTransportFromJSON(string(msg.Data))
			if err != nil {
				logger.Warnf("Could not parse received message into a TransportMessage: %s", err.Error())
				continue
			}

			reqid, federated := message.FederationRequestID()
			if !federated {
				logger.Warnf("Received a message on %s that was not federated", msg.Subject)
				continue
			}

			cm := chainmessage{
				Message:   message,
				RequestID: reqid,
				Seen:      []string{nc.ConnectedServer(), fmt.Sprintf("%s:%d", self.Name(), i)},
			}

			logger.Debugf("Received message %s via %s", reqid, message.SenderID())

			self.out <- cm
		}

	})

	return worker, err
}
