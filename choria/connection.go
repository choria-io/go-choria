package choria

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/choria-io/go-choria/tlssetup"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

// ConnectionManager is capable of being a factory for connection, mcollective.Choria is one
type ConnectionManager interface {
	NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *log.Entry) (conn Connector, err error)
}

// AgentConnector provides the minimal Connector features for subscribing and unsubscribing agents
type AgentConnector interface {
	ConnectedServer() string
	ConnectionOptions() nats.Options
	ConnectionStats() nats.Statistics
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *ConnectorMessage) error
	Unsubscribe(name string) error
	AgentBroadcastTarget(collective string, agent string) string
	ServiceBroadcastTarget(collective string, agent string) string
}

type ConnectorInfo interface {
	ConnectedServer() string
	ConnectionOptions() nats.Options
	ConnectionStats() nats.Statistics
}

// Connector is the interface a connector must implement to be valid be it NATS, Stomp, Testing etc
type Connector interface {
	AgentBroadcastTarget(collective string, agent string) string
	ChanQueueSubscribe(name string, subject string, group string, capacity int) (chan *ConnectorMessage, error)
	Close()
	Connect(ctx context.Context) (err error)
	ConnectedServer() string
	ConnectionOptions() nats.Options
	ConnectionStats() nats.Statistics
	IsConnected() bool
	Nats() *nats.Conn
	NodeDirectedTarget(collective string, identity string) string
	Publish(msg *Message) error
	PublishRaw(target string, data []byte) error
	PublishRawMsg(msg *nats.Msg) error
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *ConnectorMessage) error
	ReplyTarget(msg *Message) (string, error)
	RequestRawMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error)
	ServiceBroadcastTarget(collective string, agent string) string
	Unsubscribe(name string) error
}

type ConnectorMessage struct {
	Subject string
	Reply   string
	Data    []byte
	Msg     interface{}
}

type channelSubscription struct {
	subscription *nats.Subscription
	in           chan *nats.Msg
	out          chan *ConnectorMessage
	quit         chan interface{}
}

// Connection is a actual NATS connection handler, it implements Connector
type Connection struct {
	servers           func() (srvcache.Servers, error)
	name              string
	nats              *nats.Conn
	log               *log.Entry
	fw                *Framework
	config            *config.Config
	subscriptions     map[string]*nats.Subscription
	chanSubscriptions map[string]*channelSubscription
	outbox            chan *nats.Msg
	subMu             sync.Mutex
	conMu             sync.Mutex
	token             string
	uniqueId          string
}

var (
	connInitialConnectCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "choria_connector_initial_connection_attempts",
		Help: "How many connection attempts were made before a connection was established",
	})

	connReconnectCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "choria_connector_reconnections",
		Help: "Number of times the connector reconnected to the middleware",
	})

	connClosedCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "choria_connector_connection_closed",
		Help: "Number of times the connection was closed",
	})

	connErrorCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "choria_connector_errors",
		Help: "Number of times the connection encountered an error",
	})

	connInitialConnectTime = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "choria_connector_initial_connect_time",
		Help: "How long it took to establish the initial connection",
	})
)

func init() {
	prometheus.MustRegister(connInitialConnectCtr)
	prometheus.MustRegister(connReconnectCtr)
	prometheus.MustRegister(connClosedCtr)
	prometheus.MustRegister(connErrorCtr)
	prometheus.MustRegister(connInitialConnectTime)
}

// Bytes return the data in the message
func (m *ConnectorMessage) Bytes() []byte {
	return m.Data
}

// NewConnector creates a new NATS connector
//
// It will attempt to connect to the given servers and will keep trying till it manages to do so
func (fw *Framework) NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *log.Entry) (Connector, error) {
	if name == "" {
		name = fw.Config.Identity
	}

	conn := &Connection{
		name:              name,
		servers:           servers,
		log:               logger.WithField("connection", name),
		fw:                fw,
		config:            fw.Config,
		subscriptions:     make(map[string]*nats.Subscription),
		chanSubscriptions: make(map[string]*channelSubscription),
		outbox:            make(chan *nats.Msg, 1000),
	}

	if fw.Config.Choria.ClientAnonTLS && !fw.Config.InitiatedByServer {
		caller, id, token, err := fw.UniqueIDFromUnverifiedToken()
		if err != nil {
			return nil, fmt.Errorf("could not parse JWT: %s", err)
		}

		conn.log.Infof("Setting JWT token and unique reply queues based on JWT for %q", caller)

		conn.token = token
		conn.uniqueId = id
	}

	err := conn.Connect(ctx)

	return conn, err
}

func (conn *Connection) ConnectionOptions() nats.Options {
	return conn.nats.Opts
}

func (conn *Connection) ConnectionStats() nats.Statistics {
	return conn.nats.Statistics
}

func (conn *Connection) Nats() *nats.Conn {
	return conn.nats
}

// ChanQueueSubscribe creates a channel of a certain size and subscribes to a queue group.
//
// The given name would later be used should a unsubscribe be needed
func (conn *Connection) ChanQueueSubscribe(name string, subject string, group string, capacity int) (chan *ConnectorMessage, error) {
	var err error

	s := &channelSubscription{
		in:   make(chan *nats.Msg, capacity),
		out:  make(chan *ConnectorMessage, capacity),
		quit: make(chan interface{}, 1),
	}

	conn.subMu.Lock()
	conn.chanSubscriptions[name] = s
	conn.subMu.Unlock()

	copier := func(subs *channelSubscription) {
		for {
			select {
			case m := <-subs.in:
				subs.out <- &ConnectorMessage{Data: m.Data, Reply: m.Reply, Subject: m.Subject, Msg: m}
			case <-subs.quit:
				return
			}
		}
	}

	go copier(s)

	conn.log.Debugf("Subscribing to %s in group '%s' on server %s", subject, group, conn.ConnectedServer())

	s.subscription, err = conn.nats.ChanQueueSubscribe(subject, group, s.in)
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to subscription %s: %s", name, err)
	}

	err = conn.nats.Flush()
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to subscription %s: %s", name, err)
	}

	return s.out, nil
}

// QueueSubscribe is a lot like ChanQueueSubscribe but you provide it the queue to dump messages in,
// it also takes a context and will unsubscribe when the context is canceled
func (conn *Connection) QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *ConnectorMessage) error {
	var err error

	s := &channelSubscription{
		in:   make(chan *nats.Msg, cap(output)),
		out:  output,
		quit: make(chan interface{}, 1),
	}

	conn.subMu.Lock()
	conn.chanSubscriptions[name] = s
	conn.subMu.Unlock()

	copier := func(ctx context.Context, s *channelSubscription) {
		for {
			select {
			case m := <-s.in:
				s.out <- &ConnectorMessage{Data: m.Data, Reply: m.Reply, Subject: m.Subject, Msg: m}
			case <-ctx.Done():
				conn.Unsubscribe(name)
				return
			case <-s.quit:
				close(s.in)
				return
			}
		}
	}

	go copier(ctx, s)

	conn.log.Debugf("Subscribing to %s in group '%s' on server %s", subject, group, conn.ConnectedServer())

	s.subscription, err = conn.nats.ChanQueueSubscribe(subject, group, s.in)
	if err != nil {
		return fmt.Errorf("could not subscribe to subscription %s: %s", name, err)
	}

	err = conn.nats.Flush()
	if err != nil {
		return fmt.Errorf("could not subscribe to subscription %s: %s", name, err)
	}

	return nil
}

func (conn *Connection) Unsubscribe(name string) error {
	conn.subMu.Lock()
	defer conn.subMu.Unlock()

	if sub, ok := conn.subscriptions[name]; ok {
		err := sub.Unsubscribe()
		if err != nil {
			return fmt.Errorf("could not unsubscribe from %s: %s", name, err)
		}
	}

	if sub, ok := conn.chanSubscriptions[name]; ok {
		err := sub.subscription.Unsubscribe()
		if err != nil {
			return fmt.Errorf("could not unsubscribe from %s: %s", name, err)
		}

		sub.quit <- true

		delete(conn.chanSubscriptions, name)
	}

	return nil
}

// PublishRaw allows any data to be published to any target
func (conn *Connection) PublishRaw(target string, data []byte) error {
	conn.log.Debugf("Publishing %d bytes to %s", len(data), target)

	return conn.nats.Publish(target, data)
}

// PublishRawMsg allows any nats message to be published to any target
func (conn *Connection) PublishRawMsg(msg *nats.Msg) error {
	conn.log.Debugf("Publishing %d bytes to %s", len(msg.Data), msg.Subject)
	return conn.nats.PublishMsg(msg)
}

// RequestRawMsgWithContext allows any nats message to be published as a request
func (conn *Connection) RequestRawMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	conn.log.Debugf("Performing NATS request of %d bytes to %s", len(msg.Data), msg.Subject)
	return conn.nats.RequestMsgWithContext(ctx, msg)
}

// Publish inspects a Message and publish it according to its Type
func (conn *Connection) Publish(msg *Message) error {
	transport, err := msg.Transport()
	if err != nil {
		return err
	}

	transport.RecordNetworkHop(conn.ConnectedServer(), conn.config.Identity, conn.ConnectedServer())

	if msg.CustomTarget != "" {
		return conn.publishConnectedBroadcast(msg, transport)
	}

	if conn.fw.IsFederated() {
		return conn.publishFederated(msg, transport)
	}

	return conn.publishConnected(msg, transport)
}

func (conn *Connection) publishFederated(msg *Message, transport protocol.TransportMessage) error {
	if msg.Type() == DirectRequestMessageType {
		return conn.publishFederatedDirect(msg, transport)
	}

	return conn.publishFederatedBroadcast(msg, transport)
}

func (conn *Connection) publishFederatedDirect(msg *Message, transport protocol.TransportMessage) error {
	var err error

	util.SliceGroups(msg.DiscoveredHosts, 200, func(nodes []string) {
		targets := []string{}
		var j string

		for i := range nodes {
			if nodes[i] == "" {
				break
			}

			targets = append(targets, conn.NodeDirectedTarget(msg.collective, nodes[i]))
		}

		transport.SetFederationRequestID(msg.RequestID)
		transport.SetFederationTargets(targets)

		j, err = transport.JSON()
		if err != nil {
			return
		}

		for _, federation := range conn.fw.FederationCollectives() {
			target := conn.federationTarget(federation, "federation")

			conn.log.Debugf("Sending a federated direct message to NATS target '%s' for message %s with type %s", target, msg.RequestID, msg.Type())

			err = conn.PublishRaw(target, []byte(j))
			if err != nil {
				return
			}
		}
	})

	return err
}

func (conn *Connection) publishFederatedBroadcast(msg *Message, transport protocol.TransportMessage) error {
	target, err := conn.TargetForMessage(msg, "")
	if err != nil {
		return err
	}

	transport.SetFederationRequestID(msg.RequestID)
	transport.SetFederationTargets([]string{target})

	j, err := transport.JSON()
	if err != nil {
		return err
	}

	for _, federation := range conn.fw.FederationCollectives() {
		target := conn.federationTarget(federation, "federation")

		conn.log.Debugf("Sending a federated broadcast message to NATS target '%s' for message %s with type %s", target, msg.RequestID, msg.Type())

		msg.NotifyPublish()

		err = conn.PublishRaw(target, []byte(j))
		if err != nil {
			return err
		}
	}

	return nil
}

func (conn *Connection) publishConnected(msg *Message, transport protocol.TransportMessage) error {
	if msg.Type() == DirectRequestMessageType {
		return conn.publishConnectedDirect(msg, transport)
	}

	return conn.publishConnectedBroadcast(msg, transport)
}

func (conn *Connection) publishConnectedBroadcast(msg *Message, transport protocol.TransportMessage) error {
	j, err := transport.JSON()
	if err != nil {
		return err
	}

	target, err := conn.TargetForMessage(msg, "")
	if err != nil {
		return err
	}

	conn.log.Debugf("Sending a broadcast message to NATS target '%s' for message %s type %s", target, msg.RequestID, msg.Type())

	msg.NotifyPublish()

	err = conn.PublishRaw(target, []byte(j))
	if err != nil {
		return err
	}

	conn.Flush()

	return nil
}

func (conn *Connection) publishConnectedDirect(msg *Message, transport protocol.TransportMessage) error {
	j, err := transport.JSON()
	if err != nil {
		return err
	}

	rawmsg := []byte(j)

	for _, host := range msg.DiscoveredHosts {
		target, err := conn.TargetForMessage(msg, host)
		if err != nil {
			return fmt.Errorf("cannot publish Message %s: %s", msg.RequestID, err)
		}

		conn.log.Debugf("Sending a direct message to %s via NATS target '%s' for message %s type %s", host, target, msg.RequestID, msg.Type())

		msg.NotifyPublish()

		err = conn.PublishRaw(target, rawmsg)
		if err != nil {
			return fmt.Errorf("could not publish directed message %s to %s: %s", msg.RequestID, host, err)
		}
	}

	conn.Flush()

	return nil
}

func TargetForMessage(msg *Message, identity string) (string, error) {
	if msg.CustomTarget != "" {
		return msg.CustomTarget, nil
	}

	switch msg.Type() {
	case ReplyMessageType:
		if msg.ReplyTo() == "" {
			return "", fmt.Errorf("do not know how to reply, no reply-to header has been set on message %s", msg.RequestID)
		}

		return msg.ReplyTo(), nil

	case RequestMessageType:
		return AgentBroadcastTarget(msg.Collective(), msg.Agent), nil

	case ServiceRequestMessageType:
		return ServiceBroadcastTarget(msg.Collective(), msg.Agent), nil

	case DirectRequestMessageType:
		return NodeDirectedTarget(msg.Collective(), identity), nil

	default:
		return "", fmt.Errorf("do not know how to determine the target for Message %s with type %s", msg.RequestID, msg.Type())
	}
}

func (conn *Connection) TargetForMessage(msg *Message, identity string) (string, error) {
	return TargetForMessage(msg, identity)
}

func NodeDirectedTarget(collective string, identity string) string {
	return fmt.Sprintf("%s.node.%s", collective, identity)
}

func (conn *Connection) NodeDirectedTarget(collective string, identity string) string {
	return NodeDirectedTarget(collective, identity)
}

func AgentBroadcastTarget(collective string, agent string) string {
	return fmt.Sprintf("%s.broadcast.agent.%s", collective, agent)
}

func ServiceBroadcastTarget(collective string, agent string) string {
	return fmt.Sprintf("%s.broadcast.service.%s", collective, agent)
}

func (conn *Connection) ServiceBroadcastTarget(collective string, agent string) string {
	return ServiceBroadcastTarget(collective, agent)
}

func (conn *Connection) AgentBroadcastTarget(collective string, agent string) string {
	return AgentBroadcastTarget(collective, agent)
}

func ReplyTarget(msg *Message, requestid string) string {
	return fmt.Sprintf("%s.reply.%s.%s", msg.Collective(), fmt.Sprintf("%x", md5.Sum([]byte(msg.CallerID))), requestid)
}

func Inbox(collective string, caller string) string {
	return fmt.Sprintf("%s.reply.%s.%s", collective, fmt.Sprintf("%x", md5.Sum([]byte(caller))), util.UniqueID())
}

func (conn *Connection) ReplyTarget(msg *Message) (string, error) {
	id, err := conn.fw.NewRequestID()
	if err != nil {
		return "", err
	}

	return ReplyTarget(msg, id), nil
}

func (conn *Connection) federationTarget(federation string, side string) string {
	return fmt.Sprintf("choria.federation.%s.%s", federation, side)
}

// IsConnected determines if we are connected to the network
func (conn *Connection) IsConnected() bool {
	return conn.nats.IsConnected()
}

// ConnectedServer returns the URL of the current server that the library is connected to, "unknown" when not initialized
func (conn *Connection) ConnectedServer() string {
	if conn.Nats() == nil {
		return "unknown"
	}

	uri, err := url.Parse(conn.nats.ConnectedUrl())
	if err != nil {
		return "unknown"
	}

	return uri.String()
}

// Connect creates a new connection to NATS.
//
// This will block until connected - basically forever should it never work.  Due to short comings
// in the NATS library logging about failures is not optimal
func (conn *Connection) Connect(ctx context.Context) (err error) {
	obs := prometheus.NewTimer(connInitialConnectTime)
	defer obs.ObserveDuration()

	conn.conMu.Lock()
	defer conn.conMu.Unlock()

	options := []nats.Option{
		nats.MaxReconnects(-1),
		nats.Name(conn.name),

		// This is specifically set quite small, just about enough to handle short
		// reconnects rather than the 8MB long buffer that's default.  30 000 nodes
		// each sending several MB after reconnect is not what anyone wants
		nats.ReconnectBufSize(10 * 1024),

		nats.CustomReconnectDelay(func(n int) time.Duration {
			d := backoff.TwentySec.Duration(n)
			conn.log.Infof("Sleeping %v till the next reconnection attempt", d)

			return d
		}),

		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				conn.log.Warnf("NATS client connection got disconnected: %v", nc.LastError())
			}
		}),

		nats.ReconnectHandler(func(nc *nats.Conn) {
			conn.log.Warnf("NATS client reconnected after a previous disconnection, connected to %s", nc.ConnectedUrl())
			connReconnectCtr.Inc()
		}),

		nats.ClosedHandler(func(nc *nats.Conn) {
			err = nc.LastError()
			if err != nil {
				conn.log.Warnf("NATS client connection closed: %v", nc.LastError())
			}
			connClosedCtr.Inc()
		}),

		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			conn.log.Errorf("NATS client on %s encountered an error: %s", nc.ConnectedUrl(), err)
			connErrorCtr.Inc()
		}),
	}

	if conn.uniqueId != "" {
		options = append(options, nats.CustomInboxPrefix(fmt.Sprintf("%s.reply.%s", conn.config.MainCollective, conn.uniqueId)))
	} else {
		options = append(options, nats.CustomInboxPrefix(fmt.Sprintf("%s.reply", conn.config.MainCollective)))
	}

	if !conn.fw.Config.InitiatedByServer {
		options = append(options, nats.PingInterval(30*time.Second))
	}

	switch {
	case conn.config.Choria.ClientAnonTLS && !conn.config.InitiatedByServer:
		conn.log.Debug("Setting anonymous TLS for NATS connection")

		cfg := tlssetup.TLSConfig(conn.config)
		tlsc := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			CipherSuites:             cfg.CipherSuites,
			CurvePreferences:         cfg.CurvePreferences,
			InsecureSkipVerify:       true,
		}

		options = append(options, nats.Secure(tlsc))

		token, err := conn.fw.SignerToken()
		if err != nil {
			return fmt.Errorf("no signer token found while connecting to an anonymous TLS server: %s", err)
		}

		options = append(options, nats.Token(token))

	case !(conn.config.DisableTLS || conn.fw.ShouldUseNGS()):
		tlsc, err := conn.fw.ClientTLSConfig()
		if err != nil {
			err = fmt.Errorf("could not create TLS Config: %s", err)
			return err
		}

		options = append(options, nats.Secure(tlsc))

	default:
		conn.log.Debugf("Not specifying TLS options on NATS connection: tls: %v ngs: %v creds: %v", conn.config.DisableTLS, conn.config.Choria.NatsNGS, conn.config.Choria.NatsCredentials)
	}

	if conn.fw.ProvisionMode() && util.FileExist(conn.fw.bi.ProvisionJWTFile()) {
		t, err := os.ReadFile(conn.fw.bi.ProvisionJWTFile())
		if err == nil {
			options = append(options, nats.Token(string(t)))
		}
	}

	if !conn.config.Choria.RandomizeMiddlewareHosts {
		options = append(options, nats.DontRandomize())
	}

	if conn.config.Choria.NatsUser != "" && conn.config.Choria.NatsPass != "" {
		options = append(options, nats.UserInfo(conn.config.Choria.NatsUser, conn.config.Choria.NatsPass))
	}

	if conn.config.Choria.NatsCredentials != "" {
		options = append(options, nats.UserCredentials(conn.config.Choria.NatsCredentials))
	}

	return backoff.Default.For(ctx, func(try int) error {
		servers, err := conn.servers()
		if err != nil {
			return fmt.Errorf("could not determine servers to connect to: %s", err)
		}
		urls := strings.Join(servers.Strings(), ", ")
		conn.log.Infof("Attempting to connect to: %s", urls)
		conn.nats, err = nats.Connect(urls, options...)
		if err == nil {
			return nil
		}

		conn.log.Warnf("Initial connection to the Broker failed on try %d: %s", try, err)
		connInitialConnectCtr.Inc()

		return err
	})
}

// Flush sends any unpublished data to the network
func (conn *Connection) Flush() {
	conn.log.Debug("Flushing pending NATS messages")
	conn.nats.Flush()
}

// Close closes the NATS connection after flushing what needed to be sent
func (conn *Connection) Close() {
	subs := []string{}

	conn.subMu.Lock()

	for s := range conn.chanSubscriptions {
		subs = append(subs, s)
	}

	for s := range conn.subscriptions {
		subs = append(subs, s)
	}

	conn.subMu.Unlock()

	for _, s := range subs {
		err := conn.Unsubscribe(s)
		if err != nil {
			conn.log.Warnf("Could not unsubscribe from %s: %s", s, err)
		}
	}

	conn.Flush()

	conn.conMu.Lock()
	defer conn.conMu.Unlock()

	conn.log.Debug("Closing NATS connection")
	conn.nats.Close()
}
