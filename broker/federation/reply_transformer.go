package federation

import (
	log "github.com/sirupsen/logrus"
)

func NewChoriaReplyTransformer(workers int, capacity int, broker *FederationBroker, logger *log.Entry) (*pooledWorker, error) {
	worker, err := PooledWorkerFactory("choria_reply_transformer", workers, Unconnected, capacity, broker, logger, func(self *pooledWorker, i int, logger *log.Entry) {
		defer self.wg.Done()

		for {
			var cm chainmessage

			select {
			case cm = <-self.in:
			case <-self.done:
				logger.Infof("Worker routine %s exiting", self.Name())
				return
			}

			req, federated := cm.Message.FederationRequestID()
			if !federated {
				logger.Errorf("Received a message from %s that is not federated", cm.Message.SenderID())
				continue
			}

			replyto, _ := cm.Message.FederationReplyTo()
			if replyto == "" {
				logger.Errorf("Received message %s with no reply-to set", req)
				continue
			}

			cm.Seen = append(cm.Seen, self.Name())
			cm.Targets = []string{replyto}
			cm.RequestID = req

			cm.Message.SetUnfederated()

			logger.Infof("Received a reply message '%s' via %s", cm.RequestID, cm.Message.SenderID())

			self.out <- cm
		}

	})

	return worker, err
}
