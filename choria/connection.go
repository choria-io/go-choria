package choria

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats"
	log "github.com/sirupsen/logrus"
)

// ConnectionManager is capable of being a factory for connection, mcollective.Choria is one
type ConnectionManager interface {
	NewConnector(ervers func() ([]Server, error), name string, logger *log.Entry) (conn Connector, err error)
}

// Connector is the interface a connector must implement to be valid be it NATS, Stomp, Testing etc
type Connector interface {
	ChanQueueSubscribe(name string, subject string, group string, capacity int) (chan *ConnectorMessage, error)
	Subscribe(name string, subject string, group string) error
	Unsubscribe(name string) error

	PublishRaw(target string, data []byte) error
	Receive() *ConnectorMessage

	ConnectedServer() string
	SetServers(func() ([]Server, error))
	SetName(name string)
	Connect() (err error)
}

type ConnectorMessage struct {
	Subject string
	Reply   string
	Data    []byte
}

type channelSubscription struct {
	subscription *nats.Subscription
	in           chan *nats.Msg
	out          chan *ConnectorMessage
	quit         chan interface{}
}

// Connection is a actual NATS connectoin handler, it implements Connector
type Connection struct {
	servers           func() ([]Server, error)
	name              string
	nats              *nats.Conn
	logger            *log.Entry
	choria            *Framework
	subscriptions     map[string]*nats.Subscription
	chanSubscriptions map[string]*channelSubscription
	outbox            chan *nats.Msg
	subMu             sync.Mutex
	conMu             sync.Mutex
	recMu             sync.Mutex
}

// NewConnector creates a new NATS connector
//
// It will attempt to connect to the given servers and will keep trying till it manages to do so
func (self *Framework) NewConnector(servers func() ([]Server, error), name string, logger *log.Entry) (conn Connector, err error) {
	conn = &Connection{
		name:              name,
		servers:           servers,
		logger:            logger,
		choria:            self,
		subscriptions:     make(map[string]*nats.Subscription),
		chanSubscriptions: make(map[string]*channelSubscription),
		outbox:            make(chan *nats.Msg, 1000),
	}

	if name == "" {
		conn.SetName(self.Config.Identity)
	}

	err = conn.Connect()

	return conn, err
}

func (self *Connection) SetServers(resolver func() ([]Server, error)) {
	self.servers = resolver
}

func (self *Connection) SetName(name string) {
	self.name = name
}

// copies the incoming nats.Msg format messages on a channel subscription to its Message formatted output
func (self *Connection) copyNatstoMsg(subs *channelSubscription) {
	for {
		select {
		case m := <-subs.in:
			subs.out <- &ConnectorMessage{Data: m.Data, Reply: m.Reply, Subject: m.Subject}
		case <-subs.quit:
			return
		}
	}
}

// ChanQueueSubscribe creates a channel of a certain size and subscribes to a queue group.
//
// The given name would later be used should a unsubscribe be needed
func (self *Connection) ChanQueueSubscribe(name string, subject string, group string, capacity int) (chan *ConnectorMessage, error) {
	self.subMu.Lock()
	defer self.subMu.Unlock()

	var err error

	s := &channelSubscription{
		in:   make(chan *nats.Msg, capacity),
		out:  make(chan *ConnectorMessage, capacity),
		quit: make(chan interface{}),
	}

	self.chanSubscriptions[name] = s

	s.subscription, err = self.nats.ChanQueueSubscribe(subject, group, s.in)
	if err != nil {
		return nil, fmt.Errorf("Could not subscribe to subscription %s: %s", name, err.Error())
	}

	go self.copyNatstoMsg(s)

	return s.out, nil
}

func (self *Connection) Subscribe(name string, subject string, group string) error {
	self.subMu.Lock()
	defer self.subMu.Unlock()

	_, ok := self.subscriptions[name]
	if ok {
		return fmt.Errorf("Already have a subscription called '%s'", name)
	}

	sub, err := self.nats.ChanQueueSubscribe(subject, group, self.outbox)
	if err != nil {
		return fmt.Errorf("Could not subscribe to subscription %s: %s", name, err.Error())
	}

	self.subscriptions[name] = sub

	return nil
}

func (self *Connection) Unsubscribe(name string) error {
	self.subMu.Lock()
	defer self.subMu.Unlock()

	if sub, ok := self.subscriptions[name]; ok {
		err := sub.Unsubscribe()
		if err != nil {
			return fmt.Errorf("Could not unsubscribe from %s: %s", name, err.Error())
		}
	}

	if sub, ok := self.chanSubscriptions[name]; ok {
		err := sub.subscription.Unsubscribe()
		if err != nil {
			return fmt.Errorf("Could not unsubscribe from %s: %s", name, err.Error())
		}

		close(sub.quit)
		close(sub.in)
		close(sub.out)
		delete(self.chanSubscriptions, name)
	}

	return nil
}

// Receive waits for a message on any normal subscription as made by Subscribe()
// and pass it along, this is a blocking operation.  Subscriptions made with
// ChanQueueSubscribe is exempt from this as they all have their own queues.
//
// Only 1 instance of Recieve() can be called at any time
func (self *Connection) Receive() *ConnectorMessage {
	self.recMu.Lock()
	defer self.recMu.Unlock()

	in := <-self.outbox

	message := &ConnectorMessage{
		Data:    in.Data,
		Reply:   in.Reply,
		Subject: in.Subject,
	}

	return message
}

// PublishRaw allows any data to be published to any target
func (self *Connection) PublishRaw(target string, data []byte) error {
	return self.nats.Publish(target, data)
}

// ConnectedServer returns the URL of the current server that the library is connected to, "unknown" when not initialized
func (self *Connection) ConnectedServer() string {
	if self.nats == nil {
		return "unknown"
	}

	return self.nats.ConnectedUrl()
}

// Connect creates a new connection to NATS.
//
// This will block until connected - basically forever should it never work.  Due to short comings
// in the NATS library logging about failures is not optimal
func (self *Connection) Connect() (err error) {
	self.conMu.Lock()
	defer self.conMu.Unlock()

	var tlsc *tls.Config

	if !self.choria.Config.DisableTLS {
		tlsc, err = self.choria.TLSConfig()
		if err != nil {
			err = fmt.Errorf("Could not create TLS Config: %s", err.Error())
			return err
		}
	}

	urls := []string{}
	var url *url.URL

	servers, err := self.servers()
	if err != nil {
		err = fmt.Errorf("Could not resolve servers: %s", err.Error())
	}

	for _, server := range servers {
		url, err = server.URL()
		if err != nil {
			err = fmt.Errorf("Could not determine URL for server %#v", server)
			return
		}

		urls = append(urls, url.String())
	}

	options := []nats.Option{
		nats.MaxReconnects(-1),
		nats.Name(self.name),
		nats.DisconnectHandler(func(nc *nats.Conn) {
			self.logger.Warnf("NATS client connection got disconnected: %s", nc.LastError())
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			self.logger.Warnf("NATS client reconnected after a previous disconnection, connected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			self.logger.Warnf("NATS client connection closed: %s", nc.LastError())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			self.logger.Errorf("NATS client on %s encountered an error: %s", nc.ConnectedUrl(), err.Error())
		}),
	}

	if !self.choria.Config.DisableTLS {
		options = append(options, nats.Secure(tlsc))
	}

	for {
		self.nats, err = nats.Connect(strings.Join(urls, ", "), options...)
		if err != nil {
			self.logger.Warnf("Initial connection to the NATS broker cluster failed: %s", err.Error())
			time.Sleep(time.Second)
			continue
		}

		self.logger.Infof("Connected to %s", self.nats.ConnectedUrl())

		break
	}

	return
}
