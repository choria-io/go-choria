package federation

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/srvcache"
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

type ChoriaFramework interface {
	Configuration() *config.Config
	MiddlewareServers() (servers srvcache.Servers, err error)
	FederationMiddlewareServers() (servers srvcache.Servers, err error)
	NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *log.Entry) (conn inter.Connector, err error)
	NewRequestFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Request, err error)
	NewReplyFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Reply, err error)
	NewTransportFromJSON(data string) (message protocol.TransportMessage, err error)
}

type FederationBroker struct {
	choria ChoriaFramework

	Name string

	fedIn         connector
	fedOut        connector
	collectiveIn  connector
	collectiveOut connector

	requestT transformer
	replyT   transformer

	identity string
	logger   *log.Entry
}

func NewFederationBroker(clusterName string, choria ChoriaFramework) (broker *FederationBroker, err error) {
	broker = &FederationBroker{
		Name:     clusterName,
		choria:   choria,
		identity: choria.Configuration().Identity,
		logger:   log.WithFields(log.Fields{"cluster": clusterName, "component": "federation"}),
	}

	return
}

func (fb *FederationBroker) Start(ctx context.Context, wg *sync.WaitGroup) {
	fb.logger.Infof("Starting Federation Broker %s", fb.Name)

	defer wg.Done()

	// requests from federation
	fb.fedIn, _ = NewChoriaNatsIngest(10, Federation, 10000, fb, nil)
	fb.collectiveOut, _ = NewChoriaNatsEgest(10, Collective, 10000, fb, nil)
	fb.requestT, _ = NewChoriaRequestTransformer(10, 1000, fb, nil)
	fb.fedIn.To(fb.requestT)
	fb.requestT.To(fb.collectiveOut)

	// replies from collective
	fb.collectiveIn, _ = NewChoriaNatsIngest(10, Collective, 10000, fb, nil)
	fb.fedOut, _ = NewChoriaNatsEgest(10, Federation, 10000, fb, nil)
	fb.replyT, _ = NewChoriaReplyTransformer(10, 1000, fb, nil)
	fb.collectiveIn.To(fb.replyT)
	fb.replyT.To(fb.fedOut)

	go fb.requestT.Run(ctx)
	go fb.replyT.Run(ctx)
	go fb.collectiveOut.Run(ctx)
	go fb.collectiveIn.Run(ctx)
	go fb.requestT.Run(ctx)
	go fb.fedOut.Run(ctx)
	go fb.fedIn.Run(ctx)

	<-ctx.Done()

	log.Warn("Choria Federation Broker shutting down")
}

func nameForConnectionMode(mode int) string {
	switch mode {
	case Collective:
		return "collective"
	case Federation:
		return "federation"
	default:
		return "unconnected"
	}
}
