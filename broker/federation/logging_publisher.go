package federation

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type LoggingPublisher struct {
	chainbase
}

func (self *LoggingPublisher) Init(cluster string, instance string) error {
	self.in = make(chan chainmessage, 10000)
	self.name = fmt.Sprintf("%s:%s Logging Publisher", cluster, instance)
	self.initialized = true

	return nil
}

func (p *LoggingPublisher) Run() error {
	for {
		cm := <-p.in

		if requestid, federated := cm.Message.FederationRequestID(); federated {
			log.Infof("Received federated message %s from %s via %#v", requestid, cm.Message.SenderID(), cm.Message.SeenBy())
		} else {
			log.Infof("Received unfederated message from %s", cm.Message.SenderID())
		}
	}
}
