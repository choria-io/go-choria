// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/tokens"
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

func NewConnectorMessage(subject string, reply string, data []byte, msg any) *ConnectorMessage {
	return &ConnectorMessage{
		subject: subject,
		reply:   reply,
		data:    data,
		msg:     msg,
	}
}

type ConnectorMessage struct {
	subject string
	reply   string
	data    []byte
	msg     any
}

func (m *ConnectorMessage) Subject() string {
	return m.subject
}

func (m *ConnectorMessage) Reply() string {
	return m.reply
}

func (m *ConnectorMessage) Data() []byte {
	return m.data
}

func (m *ConnectorMessage) Msg() any {
	return m.msg
}

type channelSubscription struct {
	subscription *nats.Subscription
	in           chan *nats.Msg
	out          chan inter.ConnectorMessage
	quit         chan any
}

// Connection is a actual NATS connection handler, it implements Connector
type Connection struct {
	servers           func() (srvcache.Servers, error)
	name              string
	nats              *nats.Conn
	log               *log.Entry
	fw                *Framework
	config            *config.Config
	chanSubscriptions map[string]*channelSubscription
	outbox            chan *nats.Msg
	subMu             sync.Mutex
	conMu             sync.Mutex
	token             string
	expire            time.Time
	uniqueId          string
	ctx               context.Context
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

var (
	skipConnect bool // for testing
)

// NewConnector creates a new NATS connector
//
// It will attempt to connect to the given servers and will keep trying till it manages to do so
func (fw *Framework) NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *log.Entry) (inter.Connector, error) {
	if name == "" {
		name = fw.Config.Identity
	}

	conn := &Connection{
		name:              name,
		servers:           servers,
		log:               logger.WithField("connection", name),
		fw:                fw,
		config:            fw.Config,
		chanSubscriptions: make(map[string]*channelSubscription),
		outbox:            make(chan *nats.Msg, 1000),
		ctx:               ctx,
	}

	if conn.isJwtAuth() {
		conn.log = conn.log.WithField("jwtAuth", true)

		caller, id, exp, token, err := fw.UniqueIDFromUnverifiedToken()
		if err != nil {
			return nil, fmt.Errorf("could not parse JWT: %s", err)
		}

		// if we used such a token we would subscribe to node subject x but the broker would
		// set permissions to the identity in the token, which would not match making the node
		// unusable.
		//
		// Server run would have detected this and triggered reprovision, but we should double-check
		// here anyway and also for clients would have invalid reply channels
		if fw.Config.InitiatedByServer && caller != "" && fw.Config.Identity != caller {
			return nil, fmt.Errorf("identity %s does not match caller %s in JWT token", fw.Config.Identity, caller)
		}

		conn.log.Infof("Setting JWT token and unique reply queues based on JWT for %q (valid for %v)", caller, time.Until(exp))
		conn.token = token
		conn.expire = exp
		conn.uniqueId = id
	}

	var err error
	if !skipConnect {
		err = conn.Connect(ctx)
	}

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
func (conn *Connection) ChanQueueSubscribe(name string, subject string, group string, capacity int) (chan inter.ConnectorMessage, error) {
	var err error

	s := &channelSubscription{
		in:   make(chan *nats.Msg, capacity),
		out:  make(chan inter.ConnectorMessage, capacity),
		quit: make(chan any, 1),
	}

	conn.subMu.Lock()
	conn.chanSubscriptions[name] = s
	conn.subMu.Unlock()

	copier := func(subs *channelSubscription) {
		for {
			select {
			case m := <-subs.in:
				subs.out <- &ConnectorMessage{data: m.Data, reply: m.Reply, subject: m.Subject, msg: m}
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
func (conn *Connection) QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan inter.ConnectorMessage) error {
	var err error

	s := &channelSubscription{
		in:   make(chan *nats.Msg, cap(output)),
		out:  output,
		quit: make(chan any, 1),
	}

	conn.subMu.Lock()
	conn.chanSubscriptions[name] = s
	conn.subMu.Unlock()

	copier := func(ctx context.Context, s *channelSubscription) {
		for {
			select {
			case m := <-s.in:
				s.out <- &ConnectorMessage{data: m.Data, reply: m.Reply, subject: m.Subject, msg: m}
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

	if sub, ok := conn.chanSubscriptions[name]; ok {
		if conn.log != nil {
			conn.log.Debugf("Unsubscribing from %v", sub.subscription.Subject)
		}

		err := sub.subscription.Unsubscribe()

		// a number of errors basically mean the subscription is either
		// not there in reality, invalid or the connection is closed etc
		// so we treat all those as ok
		switch {
		case errors.Is(err, nats.ErrBadSubscription):
		case errors.Is(err, nats.ErrConnectionDraining):
		case errors.Is(err, nats.ErrConnectionClosed):
		case err != nil:
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
func (conn *Connection) Publish(msg inter.Message) error {
	transport, err := msg.Transport(conn.ctx)
	if err != nil {
		return err
	}

	transport.RecordNetworkHop(conn.ConnectedServer(), conn.config.Identity, conn.ConnectedServer())

	if msg.CustomTarget() != "" {
		return conn.publishConnectedBroadcast(msg, transport)
	}

	if conn.fw.IsFederated() {
		return conn.publishFederated(msg, transport)
	}

	return conn.publishConnected(msg, transport)
}

func (conn *Connection) publishFederated(msg inter.Message, transport protocol.TransportMessage) error {
	if msg.Type() == inter.DirectRequestMessageType {
		return conn.publishFederatedDirect(msg, transport)
	}

	return conn.publishFederatedBroadcast(msg, transport)
}

func (conn *Connection) publishFederatedDirect(msg inter.Message, transport protocol.TransportMessage) error {
	var err error

	util.SliceGroups(msg.DiscoveredHosts(), 200, func(nodes []string) {
		targets := []string{}
		var j []byte

		for i := range nodes {
			if nodes[i] == "" {
				break
			}

			targets = append(targets, conn.NodeDirectedTarget(msg.Collective(), nodes[i]))
		}

		transport.SetFederationRequestID(msg.RequestID())
		transport.SetFederationTargets(targets)

		j, err = transport.JSON()
		if err != nil {
			return
		}

		for _, federation := range conn.fw.FederationCollectives() {
			target := conn.federationTarget(federation, "federation")

			conn.log.Debugf("Sending a federated direct message to NATS target '%s' for message %s with type %s", target, msg.RequestID(), msg.Type())

			err = conn.PublishRaw(target, j)
			if err != nil {
				return
			}
		}
	})

	return err
}

func (conn *Connection) publishFederatedBroadcast(msg inter.Message, transport protocol.TransportMessage) error {
	target, err := conn.TargetForMessage(msg, "")
	if err != nil {
		return err
	}

	transport.SetFederationRequestID(msg.RequestID())
	transport.SetFederationTargets([]string{target})

	j, err := transport.JSON()
	if err != nil {
		return err
	}

	for _, federation := range conn.fw.FederationCollectives() {
		target := conn.federationTarget(federation, "federation")

		conn.log.Debugf("Sending a federated broadcast message to NATS target '%s' for message %s with type %s", target, msg.RequestID(), msg.Type())

		msg.NotifyPublish()

		err = conn.PublishRaw(target, []byte(j))
		if err != nil {
			return err
		}
	}

	return nil
}

func (conn *Connection) publishConnected(msg inter.Message, transport protocol.TransportMessage) error {
	if msg.Type() == inter.DirectRequestMessageType {
		return conn.publishConnectedDirect(msg, transport)
	}

	return conn.publishConnectedBroadcast(msg, transport)
}

func (conn *Connection) publishConnectedBroadcast(msg inter.Message, transport protocol.TransportMessage) error {
	j, err := transport.JSON()
	if err != nil {
		return err
	}

	target, err := conn.TargetForMessage(msg, "")
	if err != nil {
		return err
	}

	conn.log.Debugf("Sending a broadcast message to NATS target '%s' for message %s type %s", target, msg.RequestID(), msg.Type())

	msg.NotifyPublish()

	err = conn.PublishRaw(target, j)
	if err != nil {
		return err
	}

	conn.Flush()

	return nil
}

func (conn *Connection) publishConnectedDirect(msg inter.Message, transport protocol.TransportMessage) error {
	j, err := transport.JSON()
	if err != nil {
		return err
	}

	rawmsg := j

	for _, host := range msg.DiscoveredHosts() {
		target, err := conn.TargetForMessage(msg, host)
		if err != nil {
			return fmt.Errorf("cannot publish Message %s: %s", msg.RequestID(), err)
		}

		conn.log.Debugf("Sending a direct message to %s via NATS target '%s' for message %s type %s", host, target, msg.RequestID(), msg.Type())

		msg.NotifyPublish()

		err = conn.PublishRaw(target, rawmsg)
		if err != nil {
			return fmt.Errorf("could not publish directed message %s to %s: %s", msg.RequestID(), host, err)
		}
	}

	conn.Flush()

	return nil
}

func TargetForMessage(msg inter.Message, identity string) (string, error) {
	if msg.CustomTarget() != "" {
		return msg.CustomTarget(), nil
	}

	switch msg.Type() {
	case inter.ReplyMessageType:
		if msg.ReplyTo() == "" {
			return "", fmt.Errorf("do not know how to reply, no reply-to header has been set on message %s", msg.RequestID())
		}

		return msg.ReplyTo(), nil

	case inter.RequestMessageType:
		return AgentBroadcastTarget(msg.Collective(), msg.Agent()), nil

	case inter.ServiceRequestMessageType:
		return ServiceBroadcastTarget(msg.Collective(), msg.Agent()), nil

	case inter.DirectRequestMessageType:
		return NodeDirectedTarget(msg.Collective(), identity), nil

	default:
		return "", fmt.Errorf("do not know how to determine the target for Message %s with type %s", msg.RequestID(), msg.Type())
	}
}

func (conn *Connection) TargetForMessage(msg inter.Message, identity string) (string, error) {
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

func ReplyTarget(msg inter.Message, requestid string) string {
	// NOTE: also update msg.ReplyTarget
	return fmt.Sprintf("%s.reply.%s.%s", msg.Collective(), fmt.Sprintf("%x", md5.Sum([]byte(msg.CallerID()))), requestid)
}

func Inbox(collective string, caller string) string {
	return fmt.Sprintf("%s.reply.%s.%s", collective, fmt.Sprintf("%x", md5.Sum([]byte(caller))), util.UniqueID())
}

func (conn *Connection) ReplyTarget(msg inter.Message) (string, error) {
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

// ConnectedServer returns the URL of the current server that the library is connected to, passwords in the URL will be redacted, "unknown" when not initialized
func (conn *Connection) ConnectedServer() string {
	if conn.Nats() == nil {
		return "unknown"
	}

	uri, err := url.Parse(conn.nats.ConnectedUrlRedacted())
	if err != nil {
		return "unknown"
	}

	return uri.String()
}

func (conn *Connection) anonTLSc() *tls.Config {
	cfg := tlssetup.TLSConfig(conn.config)
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		CipherSuites:       cfg.CipherSuites,
		CurvePreferences:   cfg.CurvePreferences,
		InsecureSkipVerify: true,
	}
}

func (conn *Connection) isJwtAuth() bool {
	// in provisioning mode servers do not have seed files and the security is set to insecure, so we skip nonce signing etc for those
	return (conn.config.Choria.ServerAnonTLS || conn.config.Choria.ClientAnonTLS || conn.fw.security.BackingTechnology() == inter.SecurityTechnologyED25519JWT) && !conn.fw.ProvisionMode()
}

// Connect creates a new connection to NATS.
//
// This will block until connected - basically forever should it never work.  Due to shortcomings
// in the NATS library logging about failures is not optimal
func (conn *Connection) Connect(ctx context.Context) (err error) {
	obs := prometheus.NewTimer(connInitialConnectTime)
	defer obs.ObserveDuration()

	conn.conMu.Lock()
	defer conn.conMu.Unlock()

	options := []nats.Option{
		nats.MaxReconnects(-1),
		nats.IgnoreAuthErrorAbort(),
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
			conn.log.Warnf("NATS client reconnected after a previous disconnection, connected to %s", nc.ConnectedUrlRedacted())
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
			if nc.IsConnected() {
				conn.log.Errorf("NATS client on %s encountered an error: %s", nc.ConnectedUrlRedacted(), err)
			} else {
				conn.log.Errorf("NATS client encountered an error: %s", err)
			}

			connErrorCtr.Inc()
		}),

		nats.ConnectHandler(func(nc *nats.Conn) {
			conn.log.Infof("Connected to %s", nc.ConnectedUrlRedacted())
		}),
	}

	prefix := conn.InboxPrefix()
	if conn.uniqueId != "" {
		conn.log.Infof("Setting custom inbox prefix based on unique ID to %s", prefix)
	}
	options = append(options, nats.CustomInboxPrefix(prefix))

	if !conn.fw.Config.InitiatedByServer {
		options = append(options, nats.PingInterval(30*time.Second))
	}

	var tlsc *tls.Config

	switch {
	case conn.fw.ProvisionMode():
		if util.FileExist(conn.fw.bi.ProvisionJWTFile()) {
			t, err := os.ReadFile(conn.fw.bi.ProvisionJWTFile())
			if err == nil {
				options = append(options, nats.Token(string(t)))
			}
		}

		conn.log.Warnf("Setting anonymous TLS mode during provisioning")
		tlsc = conn.anonTLSc()

	case conn.isJwtAuth():
		conn.log.Debug("Setting JWT authentication with NONCE signatures for NATS connection")

		if conn.config.DisableTLS {
			conn.log.Warnf("Disabling TLS due to configuration override")
		} else {
			// TODO: need something better to key this off (1740)
			if conn.fw.security.BackingTechnology() == inter.SecurityTechnologyED25519JWT {
				conn.log.Debugf("Using TLS Configuration from ed25519+jwt based security system")
				tlsc, err = conn.fw.ClientTLSConfig()
				if err != nil {
					return err
				}
			} else {
				conn.log.Debugf("Configuring anonymous tls connection")
				tlsc = conn.anonTLSc()
			}
		}

		if conn.token == "" {
			return fmt.Errorf("no valid token found to sign connection NONCE")
		}
		options = append(options, nats.Token(conn.token))

		seedFile, err := conn.fw.SignerSeedFile()
		if err == nil && seedFile != "" {
			_, jwth, sigh, err := tokens.NatsConnectionHelpers(conn.token, conn.config.MainCollective, seedFile, conn.log)
			if err != nil {
				return err
			}
			options = append(options, nats.UserJWT(jwth, sigh))
		}

	case conn.config.DisableTLS:
		conn.log.Warnf("Disabling TLS due to configuration override")

	default:
		tlsc, err = conn.fw.ClientTLSConfig()
		if err != nil {
			err = fmt.Errorf("could not create TLS Config: %s", err)
			return err
		}
	}
	if tlsc != nil {
		options = append(options, nats.Secure(tlsc))
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

// InboxPrefix is the subject prefix used for replies
func (conn *Connection) InboxPrefix() string {
	if conn.uniqueId != "" {
		return fmt.Sprintf("%s.reply.%s", conn.config.MainCollective, conn.uniqueId)
	} else {
		return fmt.Sprintf("%s.reply", conn.config.MainCollective)
	}
}

// Close closes the NATS connection after flushing what needed to be sent
func (conn *Connection) Close() {
	subs := []string{}

	conn.subMu.Lock()
	for s := range conn.chanSubscriptions {
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
