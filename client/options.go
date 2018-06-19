package client

import (
	"time"

	"github.com/sirupsen/logrus"
)

// Option configures the broadcast discovery method
type Option func(c *Client)

// OnPublishStart function to call synchronously when publishing starts
func OnPublishStart(f func()) Option {
	return func(c *Client) {
		c.startPublishCB = f
	}
}

// OnPublishFinish function to call synchronously when publishing ends
func OnPublishFinish(f func()) Option {
	return func(c *Client) {
		c.endPublishCB = f
	}
}

// Timeout sets the request timeout
func Timeout(t time.Duration) Option {
	return func(c *Client) {
		c.timeout = t
	}
}

// Receivers sets how many receiver connections should be started
func Receivers(r int) Option {
	return func(c *Client) {
		c.receivers = r
	}
}

// Log sets a specific logrus logger else a new one is made
func Log(l *logrus.Entry) Option {
	return func(c *Client) {
		c.log = l
	}
}

// Connection  Supplies a custom connection, when this is supplied
// this is the only connection that will be used for the duration
// of this client for all publishes and replies
//
// This might have severe performance impact and might even cause
// message loss, my suggestion would be to use this only when doing
// batch style messages where you expect small amounts of replies
func Connection(conn Connector) Option {
	return func(c *Client) {
		c.conn = conn
	}
}
