package federation

import (
	"context"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

const (
	Unconnected = iota
	Federation
	Collective
)

type transformer interface {
	chainable
	runable
}

type connector interface {
	chainable
	runable
}

type FederationBroker struct {
	Stats   *Stats
	choria  *choria.Framework
	statsMu sync.Mutex

	Name string

	fedIn         connector
	fedOut        connector
	collectiveIn  connector
	collectiveOut connector

	requestT transformer
	replyT   transformer

	logger *log.Entry
}

func NewFederationBroker(clusterName string, choria *choria.Framework) (broker *FederationBroker, err error) {
	broker = &FederationBroker{
		Name:   clusterName,
		choria: choria,
		logger: log.WithFields(log.Fields{"cluster": clusterName, "component": "federation"}),
		Stats: &Stats{
			ConfigFile:      &choria.Config.ConfigFile,
			StartTime:       time.Now(),
			Status:          "unknown",
			CollectiveStats: &WorkerStats{ConnectedServer: "unknown"},
			FederationStats: &WorkerStats{ConnectedServer: "unknown"},
		},
	}

	return
}

func (self *FederationBroker) Start(ctx context.Context, wg *sync.WaitGroup) {
	self.logger.Infof("Starting Federation Broker %s", self.Name)

	defer wg.Done()

	// requests from federation
	self.fedIn, _ = NewChoriaNatsIngest(10, Federation, 10000, self, nil)
	self.collectiveOut, _ = NewChoriaNatsEgest(10, Collective, 10000, self, nil)
	self.requestT, _ = NewChoriaRequestTransformer(10, 1000, self, nil)
	self.fedIn.To(self.requestT)
	self.requestT.To(self.collectiveOut)

	// replies from collective
	self.collectiveIn, _ = NewChoriaNatsIngest(10, Collective, 10000, self, nil)
	self.fedOut, _ = NewChoriaNatsEgest(10, Federation, 10000, self, nil)
	self.replyT, _ = NewChoriaReplyTransformer(10, 1000, self, nil)
	self.collectiveIn.To(self.replyT)
	self.replyT.To(self.fedOut)

	go self.requestT.Run()
	go self.replyT.Run()
	go self.collectiveOut.Run()
	go self.collectiveIn.Run()
	go self.requestT.Run()
	go self.fedOut.Run()
	self.fedIn.Run()
}
