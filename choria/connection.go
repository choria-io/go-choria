package choria

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/protocol"
	"github.com/nats-io/go-nats"
	log "github.com/sirupsen/logrus"
)

// ConnectionManager is capable of being a factory for connection, mcollective.Choria is one
type ConnectionManager interface {
	NewConnector(ctx context.Context, servers func() ([]Server, error), name string, logger *log.Entry) (conn Connector, err error)
}

// PublishableConnector provides the minimal Connector features to enable publishing of choria.Message instances
type PublishableConnector interface {
	Publish(msg *Message) error
}

// AgentConnector provides the minimal Connector features for subscribing and unsubscribing agents
type AgentConnector interface {
	ConnectorInfo

	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *ConnectorMessage) error
	Unsubscribe(name string) error
	AgentBroadcastTarget(collective string, agent string) string
}

type ConnectorInfo interface {
	ConnectedServer() string
	ConnectionOptions() nats.Options
	ConnectionStats() nats.Statistics
}

type InstanceConnector interface {
	AgentConnector
	PublishableConnector

	NodeDirectedTarget(collective string, identity string) string

	Close()
}

// Connector is the interface a connector must implement to be valid be it NATS, Stomp, Testing etc
type Connector interface {
	ChanQueueSubscribe(name string, subject string, group string, capacity int) (chan *ConnectorMessage, error)
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *ConnectorMessage) error

	Subscribe(name string, subject string, group string) error
	Unsubscribe(name string) error

	AgentBroadcastTarget(collective string, agent string) string
	ReplyTarget(msg *Message) string
	NodeDirectedTarget(collective string, identity string) string

	PublishRaw(target string, data []byte) error
	Publish(msg *Message) error

	Receive() *ConnectorMessage
	Outbox() chan *nats.Msg

	ConnectedServer() string
	ConnectionOptions() nats.Options
	ConnectionStats() nats.Statistics

	SetServers(func() ([]Server, error))
	SetName(name string)
	Connect(ctx context.Context) (err error)
	Close()

	Nats() *nats.Conn
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
	config            *Config
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
func (self *Framework) NewConnector(ctx context.Context, servers func() ([]Server, error), name string, logger *log.Entry) (conn Connector, err error) {
	conn = &Connection{
		name:              name,
		servers:           servers,
		logger:            logger,
		choria:            self,
		config:            self.Config,
		subscriptions:     make(map[string]*nats.Subscription),
		chanSubscriptions: make(map[string]*channelSubscription),
		outbox:            make(chan *nats.Msg, 1000),
	}

	if name == "" {
		conn.SetName(self.Config.Identity)
	}

	err = conn.Connect(ctx)

	return conn, err
}

func (self *Connection) ConnectionOptions() nats.Options {
	return self.nats.Opts
}

func (self *Connection) ConnectionStats() nats.Statistics {
	return self.nats.Statistics
}

func (self *Connection) SetServers(resolver func() ([]Server, error)) {
	self.servers = resolver
}

func (self *Connection) SetName(name string) {
	self.name = name
}

func (self *Connection) Nats() *nats.Conn {
	return self.nats
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

	self.logger.Debugf("Susbscribing to %s in group '%s' on server %s", subject, group, self.ConnectedServer())

	s.subscription, err = self.nats.ChanQueueSubscribe(subject, group, s.in)
	if err != nil {
		return nil, fmt.Errorf("Could not subscribe to subscription %s: %s", name, err.Error())
	}

	go self.copyNatstoMsg(s)

	return s.out, nil
}

// QueueSubscribe is a lot like ChanQueueSubscribe but you provide it the queue to dump messages in
func (self *Connection) QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *ConnectorMessage) error {
	self.subMu.Lock()
	defer self.subMu.Unlock()

	var err error

	s := &channelSubscription{
		in:   make(chan *nats.Msg, 5000),
		out:  output,
		quit: make(chan interface{}),
	}

	self.chanSubscriptions[name] = s

	self.logger.Debugf("Susbscribing to %s in group '%s' on server %s", subject, group, self.ConnectedServer())

	s.subscription, err = self.nats.ChanQueueSubscribe(subject, group, s.in)
	if err != nil {
		return fmt.Errorf("Could not subscribe to subscription %s: %s", name, err.Error())
	}

	copier := func(ctx context.Context, s *channelSubscription) {
		for {
			select {
			case m := <-s.in:
				s.out <- &ConnectorMessage{Data: m.Data, Reply: m.Reply, Subject: m.Subject}
			case <-ctx.Done():
				return
			case <-s.quit:
				return
			}
		}
	}

	go copier(ctx, s)

	return err
}

func (self *Connection) Subscribe(name string, subject string, group string) error {
	self.subMu.Lock()
	defer self.subMu.Unlock()

	_, ok := self.subscriptions[name]
	if ok {
		return fmt.Errorf("Already have a subscription called '%s'", name)
	}

	self.logger.Debugf("Susbscribing to %s in group '%s' on server %s", subject, group, self.ConnectedServer())

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

// Outbox gives access to the outbox of raw nats messages
// this is a bit of a hack for now because this connector was
// written before I knew about contexts and all sorts of things
// so this gets me going but I will soon have to rewrite this
// whole connector mess into something better
func (self *Connection) Outbox() chan *nats.Msg {
	return self.outbox
}

// PublishRaw allows any data to be published to any target
func (self *Connection) PublishRaw(target string, data []byte) error {
	log.Debugf("Publishing %d bytes to %s", len(data), target)

	return self.nats.Publish(target, data)
}

// Publish inspects a Message and publish it according to its Type
func (self *Connection) Publish(msg *Message) error {
	transport, err := msg.Transport()
	if err != nil {
		return fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
	}

	transport.RecordNetworkHop(self.ConnectedServer(), self.choria.Config.Identity, self.ConnectedServer())

	if msg.CustomTarget != "" {
		return self.publishConnectedBroadcast(msg, transport)
	}

	if self.choria.IsFederated() {
		return self.publishFederated(msg, transport)
	}

	return self.publishConnected(msg, transport)
}

func (self *Connection) publishFederated(msg *Message, transport protocol.TransportMessage) error {
	if msg.Type() == "direct_request" {
		return self.publishFederatedDirect(msg, transport)
	}

	return self.publishFederatedBroadcast(msg, transport)
}

func (self *Connection) publishFederatedDirect(msg *Message, transport protocol.TransportMessage) error {
	var err error

	SliceGroups(msg.DiscoveredHosts, 200, func(nodes []string) {
		targets := []string{}
		var j string

		for i := range nodes {
			if nodes[i] == "" {
				break
			}

			targets = append(targets, self.NodeDirectedTarget(msg.collective, nodes[i]))
		}

		transport.SetFederationRequestID(msg.RequestID)
		transport.SetFederationTargets(targets)

		j, err = transport.JSON()
		if err != nil {
			err = fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
			return
		}

		for _, federation := range self.choria.FederationCollectives() {
			target := self.federationTarget(federation, "federation")

			log.Debugf("Sending a federated direct message to NATS target '%s' for message %s with type %s", target, msg.RequestID, msg.Type())

			err = self.PublishRaw(target, []byte(j))
			if err != nil {
				err = fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
				return
			}
		}
	})

	return err
}

func (self *Connection) publishFederatedBroadcast(msg *Message, transport protocol.TransportMessage) error {
	target, err := self.TargetForMessage(msg, "")
	if err != nil {
		return fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
	}

	transport.SetFederationRequestID(msg.RequestID)
	transport.SetFederationTargets([]string{target})

	j, err := transport.JSON()
	if err != nil {
		return fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
	}

	for _, federation := range self.choria.FederationCollectives() {
		target := self.federationTarget(federation, "federation")

		log.Debugf("Sending a federated broadcast message to NATS target '%s' for message %s with type %s", target, msg.RequestID, msg.Type())

		err = self.PublishRaw(target, []byte(j))
		if err != nil {
			return fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
		}
	}

	return nil
}

func (self *Connection) publishConnected(msg *Message, transport protocol.TransportMessage) error {
	if msg.Type() == "direct_request" {
		return self.publishConnectedDirect(msg, transport)
	}

	return self.publishConnectedBroadcast(msg, transport)
}

func (self *Connection) publishConnectedBroadcast(msg *Message, transport protocol.TransportMessage) error {
	j, err := transport.JSON()
	if err != nil {
		return fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
	}

	target, err := self.TargetForMessage(msg, "")
	if err != nil {
		return fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
	}

	log.Debugf("Sending a broadcast message to NATS target '%s' for message %s type %s", target, msg.RequestID, msg.Type())

	return self.PublishRaw(target, []byte(j))

}

func (self *Connection) publishConnectedDirect(msg *Message, transport protocol.TransportMessage) error {
	j, err := transport.JSON()
	if err != nil {
		return fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
	}

	rawmsg := []byte(j)

	for _, host := range msg.DiscoveredHosts {
		target, err := self.TargetForMessage(msg, host)
		if err != nil {
			return fmt.Errorf("Cannot publish Message %s: %s", msg.RequestID, err.Error())
		}

		log.Debugf("Sending a direct message to %s via NATS target '%s' for message %s type %s", host, target, msg.RequestID, msg.Type())

		err = self.PublishRaw(target, rawmsg)
		if err != nil {
			return fmt.Errorf("Could not publish directed message %s to %s: %s", msg.RequestID, host, err.Error())
		}
	}

	return nil
}

func (self *Connection) TargetForMessage(msg *Message, identity string) (string, error) {
	if msg.CustomTarget != "" {
		return msg.CustomTarget, nil
	}

	if msg.Type() == "reply" {
		if msg.ReplyTo() == "" {
			return "", fmt.Errorf("Do not know how to reply, no reply-to header has been set on message %s", msg.RequestID)
		}

		return msg.ReplyTo(), nil

	} else if msg.Type() == "request" {
		return self.AgentBroadcastTarget(msg.Collective(), msg.Agent), nil

	} else if msg.Type() == "direct_request" {
		return self.NodeDirectedTarget(msg.Collective(), identity), nil
	}

	return "", fmt.Errorf("Do not know how to determine the target for Message %s with type %s", msg.RequestID, msg.Type())
}

func (self *Connection) NodeDirectedTarget(collective string, identity string) string {
	return fmt.Sprintf("%s.node.%s", collective, identity)
}

func (self *Connection) AgentBroadcastTarget(collective string, agent string) string {
	return fmt.Sprintf("%s.broadcast.agent.%s", collective, agent)
}

func (self *Connection) ReplyTarget(msg *Message) string {
	return fmt.Sprintf("%s.reply.%s.%s", msg.Collective(), msg.SenderID, self.choria.NewRequestID())
}

func (self *Connection) federationTarget(federation string, side string) string {
	return fmt.Sprintf("choria.federation.%s.%s", federation, side)
}

// ConnectedServer returns the URL of the current server that the library is connected to, "unknown" when not initialized
func (self *Connection) ConnectedServer() string {
	if self.Nats == nil {
		return "unknown"
	}

	url, err := url.Parse(self.nats.ConnectedUrl())
	if err != nil {
		return "unknown"
	}

	return fmt.Sprintf("nats://%s:%s", strings.TrimSuffix(url.Hostname(), "."), url.Port())
}

// Connect creates a new connection to NATS.
//
// This will block until connected - basically forever should it never work.  Due to short comings
// in the NATS library logging about failures is not optimal
func (self *Connection) Connect(ctx context.Context) (err error) {
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
			err = nc.LastError()

			if err != nil {
				self.logger.Warnf("NATS client connection got disconnected: %s", nc.LastError())
			}
		}),

		nats.ReconnectHandler(func(nc *nats.Conn) {
			self.logger.Warnf("NATS client reconnected after a previous disconnection, connected to %s", nc.ConnectedUrl())
		}),

		nats.ClosedHandler(func(nc *nats.Conn) {
			err = nc.LastError()
			if err != nil {
				self.logger.Warnf("NATS client connection closed: %s", nc.LastError())
			}
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

			if ctx.Err() != nil {
				err = fmt.Errorf("Initial connection cancelled due to shut down")
				return
			}

			time.Sleep(time.Second)
			continue
		}

		self.logger.Infof("Connected to %s", self.nats.ConnectedUrl())

		break
	}

	return
}

// Close closes the NATS connection after flushing what needed to be sent
func (self *Connection) Close() {
	for s := range self.chanSubscriptions {
		self.logger.Debugf("Stopping channel subscription %s", s)
		self.chanSubscriptions[s].quit <- true
	}

	self.logger.Debug("Flushing pending NATS messages before close")
	self.nats.Flush()

	self.logger.Debug("Closing NATS connection")
	self.nats.Close()
}
