package federation

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/statistics"
	log "github.com/sirupsen/logrus"
)

func NewChoriaRequestTransformer(workers int, capacity int, broker *FederationBroker, logger *log.Entry) (*pooledWorker, error) {
	worker, err := PooledWorkerFactory("choria_request_transformer", workers, Unconnected, capacity, broker, logger, func(ctx context.Context, self *pooledWorker, i int, logger *log.Entry) {
		defer self.wg.Done()

		rctr := statistics.Counter(fmt.Sprintf("federation.choria_request_transformer.%d.received", i))
		ectr := statistics.Counter(fmt.Sprintf("federation.choria_request_transformer.%d.err", i))
		timer := statistics.Timer(fmt.Sprintf("federation.choria_request_transformer.%d.time", i))

		for {
			var cm chainmessage

			select {
			case cm = <-self.in:
			case <-ctx.Done():
				logger.Infof("Worker routine %s exiting", self.Name())
				return
			}

			req, federated := cm.Message.FederationRequestID()
			if !federated {
				logger.Errorf("Received a message from %s that is not federated", cm.Message.SenderID())
				ectr.Inc(1)
				continue
			}

			targets, _ := cm.Message.FederationTargets()
			if len(targets) == 0 {
				logger.Errorf("Received a message %s from %s that does not have any targets", req, cm.Message.SenderID())
				ectr.Inc(1)
				continue
			}

			replyto := cm.Message.ReplyTo()
			if replyto == "" {
				logger.Errorf("Received a message %s with no reply-to set", req)
				ectr.Inc(1)
				continue
			}

			timer.Time(func() {
				cm.Seen = append(cm.Seen, fmt.Sprintf("%s:%d", self.Name(), i))
				cm.RequestID = req
				cm.Targets = targets

				cm.Message.SetFederationTargets([]string{})
				cm.Message.SetFederationReplyTo(replyto)
				cm.Message.SetReplyTo(fmt.Sprintf("choria.federation.%s.collective", self.broker.Name))

				logger.Infof("Received request message '%s' via %s with %d targets", cm.RequestID, cm.Message.SenderID(), len(cm.Targets))

				self.out <- cm
			})

			rctr.Inc(1)
		}
	})

	return worker, err
}
