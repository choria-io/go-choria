package federation

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/broker/federation/stats"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func NewChoriaReplyTransformer(workers int, capacity int, broker *FederationBroker, logger *log.Entry) (*pooledWorker, error) {
	worker, err := PooledWorkerFactory("choria_reply_transformer", workers, Unconnected, capacity, broker, logger, func(ctx context.Context, self *pooledWorker, i int, logger *log.Entry) {
		defer self.wg.Done()

		workeri := fmt.Sprintf("%d", i)
		rctr := stats.ReceivedMsgsCtr.WithLabelValues("choria_reply_transformer", workeri, nameForConnectionMode(Unconnected), self.broker.Name, self.broker.identity)
		ectr := stats.ErrorCtr.WithLabelValues("choria_reply_transformer", workeri, nameForConnectionMode(Unconnected), self.broker.Name, self.broker.identity)
		timer := stats.ProcessTime.WithLabelValues("choria_reply_transformer", workeri, nameForConnectionMode(Unconnected), self.broker.Name, self.broker.identity)

		transf := func(cm chainmessage) {
			obs := prometheus.NewTimer(timer)
			defer obs.ObserveDuration()

			req, federated := cm.Message.FederationRequestID()
			if !federated {
				logger.Errorf("Received a message from %s that is not federated", cm.Message.SenderID())
				ectr.Inc()
				return
			}

			replyto, _ := cm.Message.FederationReplyTo()
			if replyto == "" {
				logger.Errorf("Received message %s with no reply-to set", req)
				ectr.Inc()
				return
			}

			cm.Seen = append(cm.Seen, fmt.Sprintf("%s:%d", self.Name(), i))
			cm.Targets = []string{replyto}
			cm.RequestID = req

			cm.Message.SetUnfederated()

			logger.Infof("Received a reply message '%s' via %s", cm.RequestID, cm.Message.SenderID())

			self.out <- cm

			rctr.Inc()
		}

		for {
			var cm chainmessage

			select {
			case cm = <-self.in:
				transf(cm)
			case <-ctx.Done():
				logger.Infof("Worker routine %s exiting", self.Name())
				return
			}

		}
	})

	return worker, err
}
