package federation

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/mcollective"
)

// Publisher implements a publisher to some middleware
type connector interface {
	chainable
	runable
}

// Transformer transforms a message format by adding headers or rewriting it etc
type transformer interface {
	chainable
	runable
}

type FederationBroker struct {
	Stats   *Stats
	statsMu sync.Mutex

	clusterName  string
	instanceName string

	fedIn         connector
	fedOut        connector
	collectiveIn  connector
	collectiveOut connector

	requestT transformer
	replyT   transformer
}

func NewFederationBroker(clusterName string, instanceName string, choria *mcollective.Choria) (broker *FederationBroker, err error) {
	broker = &FederationBroker{
		clusterName:  clusterName,
		instanceName: instanceName,
		Stats: &Stats{
			ConfigFile:      &choria.Config.ConfigFile,
			StartTime:       time.Now(),
			Status:          "unknown",
			CollectiveStats: &WorkerStats{ConnectedServer: "unknown"},
			FederationStats: &WorkerStats{ConnectedServer: "unknown"},
		},
	}

	broker.initReplyTransformer()
	broker.initRequestTransformer()

	return
}

func (self *FederationBroker) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	self.fedIn = &RequestGenerator{}
	self.fedIn.Init(self.clusterName, self.instanceName)

	self.fedOut = &LoggingPublisher{}
	self.fedOut.Init(self.clusterName, self.instanceName)

	self.fedIn.To(self.requestT)
	self.requestT.To(self.fedOut)

	self.startRequestTransformer()
	go self.fedOut.Run()
	go self.fedIn.Run()
}

func (self *FederationBroker) initRequestTransformer() (err error) {
	if self.requestT == nil {
		self.requestT = &RequestTransformer{}
		if err = self.requestT.Init(self.clusterName, self.instanceName); err != nil {
			err = fmt.Errorf("RequestTransformer initialization failed: %s", err.Error())
			return
		}

	}

	if !self.requestT.Ready() {
		err = errors.New("RequestTransformer did not become Ready after initialization")
	}

	return
}

func (self *FederationBroker) initReplyTransformer() (err error) {
	if self.replyT == nil {
		self.replyT = &ReplyTransformer{}
		if err = self.replyT.Init(self.clusterName, self.instanceName); err != nil {
			err = fmt.Errorf("ReplyTransformer initialization failed: %s", err.Error())
			return
		}

	}

	if !self.replyT.Ready() {
		err = errors.New("ReplyTransformer did not become Ready after initialization")
	}

	return
}

func (self *FederationBroker) startRequestTransformer() (err error) {
	if err = self.initRequestTransformer(); err != nil {
		err = fmt.Errorf("Could not initialize: %s", err.Error())
		return
	}

	go self.requestT.Run()

	return
}

func (self *FederationBroker) startReplyTransformer() (err error) {
	if err = self.initReplyTransformer(); err != nil {
		err = fmt.Errorf("Could not initialize: %s", err.Error())
		return
	}

	go self.replyT.Run()

	return
}
