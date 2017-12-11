package federation

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/statistics"
	log "github.com/sirupsen/logrus"
)

func NewChoriaReplyTransformer(workers int, capacity int, broker *FederationBroker, logger *log.Entry) (*pooledWorker, error) {
	worker, err := PooledWorkerFactory("choria_reply_transformer", workers, Unconnected, capacity, broker, logger, func(ctx context.Context, self *pooledWorker, i int, logger *log.Entry) {
		defer self.wg.Done()

		rctr := statistics.Counter(fmt.Sprintf("federation.choria_reply_transformer.%d.received", i))
		ectr := statistics.Counter(fmt.Sprintf("federation.choria_reply_transformer.%d.err", i))
		timer := statistics.Timer(fmt.Sprintf("federation.choria_reply_transformer.%d.time", i))

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

			replyto, _ := cm.Message.FederationReplyTo()
			if replyto == "" {
				logger.Errorf("Received message %s with no reply-to set", req)
				ectr.Inc(1)
				continue
			}

			timer.Time(func() {
				cm.Seen = append(cm.Seen, fmt.Sprintf("%s:%d", self.Name(), i))
				cm.Targets = []string{replyto}
				cm.RequestID = req

				cm.Message.SetUnfederated()

				logger.Infof("Received a reply message '%s' via %s", cm.RequestID, cm.Message.SenderID())

				self.out <- cm
			})

			rctr.Inc(1)
		}
	})

	return worker, err
}
