package federation

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

// RequestTransformer transforms federated requests from the Federation into the format that
// will be published to the Collective
type RequestTransformer struct {
	chainbase

	// The number of worker go procs that will be created to consume requests from the channel.  Defaults 10
	Workers int
}

func (self *RequestTransformer) process(cm chainmessage, name string) (chainmessage, error) {
	req, federated := cm.Message.FederationRequestID()
	if !federated {
		return chainmessage{}, fmt.Errorf("%s received a message from %s that is not federated", name, cm.Message.SenderID())
	}

	targets, _ := cm.Message.FederationTargets()
	if len(targets) == 0 {
		return chainmessage{}, fmt.Errorf("%s received a message %s from %s that does not have any targets", name, req, cm.Message.SenderID())
	}

	replyto := cm.Message.ReplyTo()
	if replyto == "" {
		return chainmessage{}, fmt.Errorf("%s received a message %s with no reply-to set", name, req)
	}

	cm.RequestID = req
	cm.Targets = targets
	cm.Message.SetFederationTargets([]string{})
	cm.Message.SetFederationReplyTo(replyto)
	cm.Message.SetReplyTo("federation.reply.target") // TODO

	return cm, nil
}

func (self *RequestTransformer) processor(instance int, wg *sync.WaitGroup) {
	defer wg.Done()

	name := fmt.Sprintf("%s:%d", self.Name(), instance)

	for {
		cm, err := self.process(<-self.in, name)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		log.Infof("%s received request message %s", name, cm.RequestID)

		self.out <- cm
	}
}

func (self *RequestTransformer) Run() error {
	wg := sync.WaitGroup{}

	for i := 0; i < self.Workers; i++ {
		wg.Add(1)
		go self.processor(i, &wg)
	}

	wg.Wait()

	return nil
}

func (self *RequestTransformer) Init(cluster string, instance string) error {
	self.in = make(chan chainmessage, 1000)
	self.out = make(chan chainmessage, 1000)
	self.name = fmt.Sprintf("%s:%s choria_request_transformer", cluster, instance)

	if self.Workers == 0 {
		self.Workers = 10
	}

	log.Infof("Initialized %s with %d workers", self.name, self.Workers)

	self.initialized = true

	return nil
}
