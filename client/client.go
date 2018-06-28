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
	"github.com/choria-io/go-choria/srvcache"
	"github.com/sirupsen/logrus"
)

// Client is a basic low level high performance Choria client
type Client struct {
	ctx           context.Context
	cancel        func()
	fw            *choria.Framework
	wg            *sync.WaitGroup
	receiverReady chan struct{}
	replies       chan *choria.ConnectorMessage
	timeout       time.Duration
	conn          Connector
	receivers     int
	log           *logrus.Entry

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
func New(fw *choria.Framework, opts ...Option) (*Client, error) {
	c := &Client{
		fw:        fw,
		wg:        &sync.WaitGroup{},
		receivers: 1,
		log:       fw.Logger("client"),
		replies:   make(chan *choria.ConnectorMessage, 100000),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.timeout == 0 {
		c.timeout = time.Duration(fw.Config.DiscoveryTimeout) * time.Second
	}

	if c.receivers <= 0 {
		return nil, errors.New("receivers should be more than 1")
	}

	c.receiverReady = make(chan struct{}, c.receivers)

	return c, nil
}

// Request performs a request
//
// `handler`` will  be called for every reply that gets received, when `handler`
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

		name := fmt.Sprintf("%s_%s", c.fw.Certname(), msg.RequestID)

		for i := 0; i < c.receivers; i++ {
			c.wg.Add(1)
			go c.receiver(name, i, msg.ReplyTo(), handler)
		}
	} else {
		c.receiverReady <- struct{}{}
	}

	c.wg.Add(1)
	go c.publish(msg)

	c.wg.Wait()

	return nil
}

func (c *Client) publish(msg *choria.Message) {
	defer c.wg.Done()

	conn := c.conn
	var err error

	if conn == nil {
		conn, err = c.connect(fmt.Sprintf("%s-%s-publisher", c.fw.Certname(), c.fw.NewRequestID()))
		if err != nil {
			c.log.Errorf("could not connect: %s", err)
		}
	}

	select {
	case <-c.receiverReady:
	case <-c.ctx.Done():
		return
	}

	if c.startPublishCB != nil {
		c.startPublishCB()
	}

	// TODO needs context https://github.com/choria-io/go-choria/issues/211
	err = conn.Publish(msg)
	if err != nil {
		c.log.Error(err)
		return
	}

	if c.endPublishCB != nil {
		c.endPublishCB()
	}

	return
}

func (c *Client) receiver(name string, i int, target string, cb Handler) {
	defer c.wg.Done()

	conn := c.conn
	var err error

	if conn == nil {
		conn, err = c.connect(fmt.Sprintf("%s-receiver%d", name, i))
		if err != nil {
			c.log.Errorf("could not connect: %s", err)
			return
		}
	}

	c.wg.Add(1)
	go c.msgHandler(cb)

	grp := ""
	if c.receivers > 1 {
		grp = name
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
	servers := func() ([]srvcache.Server, error) {
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
