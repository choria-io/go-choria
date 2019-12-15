// Package client is a low level client to the Choria network
//
// It is capable of publishing any raw data contained in a choria Message
// to the network and supports federations, SRV records and everything else
//
// This client has no awareness of the RPC system or anything like that, it's
// the lowest level raw access to the network from which higher order abstractions
// can be made like those that the RPC libraries require or discovery systems
package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-config"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/choria-io/go-srvcache"
	"github.com/sirupsen/logrus"
)

type ChoriaFramework interface {
	Configuration() *config.Config
	Logger(string) *logrus.Entry
	NewRequestID() (string, error)
	Certname() string
	MiddlewareServers() (srvcache.Servers, error)
	NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *logrus.Entry) (conn choria.Connector, err error)
	NewMessage(payload string, agent string, collective string, msgType string, request *choria.Message) (msg *choria.Message, err error)
	NewTransportFromJSON(data string) (message protocol.TransportMessage, err error)
}

// Client is a basic low level high performance Choria client
type Client struct {
	ctx           context.Context
	cancel        func()
	fw            ChoriaFramework
	cfg           *config.Config
	wg            *sync.WaitGroup
	receiverReady chan struct{}
	replies       chan *choria.ConnectorMessage
	timeout       time.Duration
	conn          Connector
	receivers     int
	log           *logrus.Entry
	name          string

	startPublishCB func()
	endPublishCB   func()
}

// Handler handles individual messages
type Handler func(ctx context.Context, m *choria.ConnectorMessage)

// Connector is a connection to the choria network
type Connector interface {
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *choria.ConnectorMessage) error
	Publish(msg *choria.Message) error
}

// New creates a Choria client
func New(fw ChoriaFramework, opts ...Option) (*Client, error) {
	c := &Client{
		fw:        fw,
		cfg:       fw.Configuration(),
		wg:        &sync.WaitGroup{},
		receivers: 1,
		log:       fw.Logger("client"),
		replies:   make(chan *choria.ConnectorMessage, 100000),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.timeout == 0 {
		c.timeout = time.Duration(c.cfg.DiscoveryTimeout) * time.Second
	}

	if c.receivers <= 0 {
		return nil, errors.New("receivers should be more than 1")
	}

	if c.name == "" {
		rid, err := c.fw.NewRequestID()
		if err != nil {
			return nil, fmt.Errorf("could not generate unique name: %s", err)
		}

		c.name = fmt.Sprintf("%s-%s", c.fw.Certname(), rid)
	}

	c.receiverReady = make(chan struct{}, c.receivers)

	return c, nil
}

// Request performs a request
//
// handler will  be called for every reply that gets received, when handler
// is nil this means no receiving listeners, workers or subscriptions are setup
// effectively the message is published and forgotten
//
// This fire and forget approach is useful when one do not care for the replies
// or when the reply to target in the message is set to a custom reply target
// meaning the client will anyway never receive the replies
func (c *Client) Request(ctx context.Context, msg *choria.Message, handler Handler) (err error) {
	// will be used later to handle shutting everything down when a maximum wait for messages
	// was processed
	c.ctx, c.cancel = context.WithCancel(ctx)
	defer c.cancel()

	if handler != nil {
		c.log.Debugf("Starting %d receivers on %s", c.receivers, msg.ReplyTo())

		for i := 0; i < c.receivers; i++ {
			c.wg.Add(1)
			go c.receiver(i, msg.ReplyTo(), handler)
		}
	} else {
		c.receiverReady <- struct{}{}
	}

	err = c.publish(msg)
	if err != nil {
		return err
	}

	c.wg.Wait()

	return err
}

func (c *Client) publish(msg *choria.Message) error {
	conn := c.conn
	var err error

	if conn == nil {
		conn, err = c.connect(fmt.Sprintf("%s-publisher", c.name))
		if err != nil {
			return fmt.Errorf("could not connect: %s", err)
		}
	}

	select {
	case <-c.receiverReady:
	case <-c.ctx.Done():
		return nil
	}

	if c.startPublishCB != nil {
		c.startPublishCB()
	}

	if c.endPublishCB != nil {
		defer c.endPublishCB()
	}

	// TODO needs context https://github.com/choria-io/go-choria/issues/211
	err = conn.Publish(msg)
	if err != nil {
		return fmt.Errorf("could not publish request: %s", err)
	}

	return nil
}

func (c *Client) receiver(i int, target string, cb Handler) {
	defer c.wg.Done()

	conn := c.conn
	var err error

	if conn == nil {
		conn, err = c.connect(fmt.Sprintf("%s-receiver%d", c.name, i))
		if err != nil {
			c.log.Errorf("could not connect: %s", err)
			return
		}
	}

	c.wg.Add(1)
	go c.msgHandler(cb)

	grp := ""
	if c.receivers > 1 {
		grp = c.name
	}

	conn.QueueSubscribe(c.ctx, "replies", target, grp, c.replies)

	c.receiverReady <- struct{}{}
}

func (c *Client) msgHandler(cb Handler) {
	defer c.wg.Done()

	timeout := time.After(c.timeout)

	for {
		select {
		case rawmsg := <-c.replies:
			cb(c.ctx, rawmsg)

		case <-c.ctx.Done():
			return

		case <-timeout:
			c.log.Debugf("Timeout while waiting for message")
			return
		}
	}
}

func (c *Client) connect(name string) (Connector, error) {
	servers := func() (srvcache.Servers, error) {
		return c.fw.MiddlewareServers()
	}

	connector, err := c.fw.NewConnector(c.ctx, servers, name, c.log)
	if err != nil {
		return nil, fmt.Errorf("could not create connector: %s", err)
	}

	closer := func() {
		select {
		case <-c.ctx.Done():
			c.log.Debug("Closing connection")
			connector.Close()
			c.conn = nil
		}
	}

	go closer()

	return connector, nil
}
