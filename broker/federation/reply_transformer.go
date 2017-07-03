package federation

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

// ReplyTransformer transforms federated replies from the Collective into the format that
// will be published to the Federation
type ReplyTransformer struct {
	chainbase

	Workers int
}

func (self *ReplyTransformer) process(cm chainmessage, name string) (chainmessage, error) {
	req, federated := cm.Message.FederationRequestID()
	if !federated {
		return chainmessage{}, fmt.Errorf("%s received a message from %s that is not federated", name, cm.Message.SenderID())
	}

	replyto, _ := cm.Message.FederationReplyTo()
	if replyto == "" {
		return chainmessage{}, fmt.Errorf("%s received a message %s with no reply-to set", name, req)
	}

	cm.Targets = []string{replyto}
	cm.RequestID = req

	cm.Message.SetUnfederated()

	return cm, nil
}

func (self *ReplyTransformer) processor(instance int, wg *sync.WaitGroup) {
	defer wg.Done()

	name := fmt.Sprintf("%s:%d", self.Name(), instance)

	for {
		cm, err := self.process(<-self.in, name)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		log.Info("%s received a reply message %s", name, cm.RequestID)

		self.out <- cm
	}
}

func (self *ReplyTransformer) Run() error {
	wg := sync.WaitGroup{}

	for i := 0; i < self.Workers; i++ {
		wg.Add(1)
		go self.processor(i, &wg)
	}

	wg.Wait()

	return nil
}

func (self *ReplyTransformer) Init(cluster string, instance string) error {
	self.in = make(chan chainmessage, 1000)
	self.out = make(chan chainmessage, 1000)
	self.name = fmt.Sprintf("%s:%s choria_reply_transformer", cluster, instance)

	if self.Workers == 0 {
		self.Workers = 10
	}

	log.Infof("Initialized %s with %d workers", self.name, self.Workers)

	self.initialized = true

	return nil
}
