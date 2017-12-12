package federation

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/broker/federation/stats"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func NewChoriaRequestTransformer(workers int, capacity int, broker *FederationBroker, logger *log.Entry) (*pooledWorker, error) {
	worker, err := PooledWorkerFactory("choria_request_transformer", workers, Unconnected, capacity, broker, logger, func(ctx context.Context, self *pooledWorker, i int, logger *log.Entry) {
		defer self.wg.Done()

		workeri := fmt.Sprintf("%d", i)
		rctr := stats.ReceivedMsgsCtr.WithLabelValues("choria_request_transformer", workeri, "")
		ectr := stats.ErrorCtr.WithLabelValues("choria_request_transformer", workeri, "")
		timer := stats.ProcessTime.WithLabelValues("choria_request_transformer", workeri, "")

		workerf := func(cm chainmessage) {
			obs := prometheus.NewTimer(timer)
			defer obs.ObserveDuration()

			req, federated := cm.Message.FederationRequestID()
			if !federated {
				logger.Errorf("Received a message from %s that is not federated", cm.Message.SenderID())
				ectr.Inc()
				return
			}

			targets, _ := cm.Message.FederationTargets()
			if len(targets) == 0 {
				logger.Errorf("Received a message %s from %s that does not have any targets", req, cm.Message.SenderID())
				ectr.Inc()
				return
			}

			replyto := cm.Message.ReplyTo()
			if replyto == "" {
				logger.Errorf("Received a message %s with no reply-to set", req)
				ectr.Inc()
				return
			}

			cm.Seen = append(cm.Seen, fmt.Sprintf("%s:%d", self.Name(), i))
			cm.RequestID = req
			cm.Targets = targets

			cm.Message.SetFederationTargets([]string{})
			cm.Message.SetFederationReplyTo(replyto)
			cm.Message.SetReplyTo(fmt.Sprintf("choria.federation.%s.collective", self.broker.Name))

			logger.Infof("Received request message '%s' via %s with %d targets", cm.RequestID, cm.Message.SenderID(), len(cm.Targets))

			self.out <- cm

			rctr.Inc()
		}

		for {
			var cm chainmessage

			select {
			case cm = <-self.in:
				workerf(cm)
			case <-ctx.Done():
				logger.Infof("Worker routine %s exiting", self.Name())
				return
			}

		}
	})

	return worker, err
}
