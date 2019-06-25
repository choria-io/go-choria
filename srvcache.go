// Package srvcache provides a caching SRV lookup service that creates a short term
// cache of SRV answers - it does not comply with DNS protocols like the timings
// set by DNS servers, its more a short term - think 5 seconds - buffer to avoid
// hitting the dns servers repeatedly
package srvcache

import (
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// Servers is a collection of server urls
type Servers interface {
	Count() int
	Strings() (urls []string)
	URLs() (urls []*url.URL, err error)
	HostPorts() (hps []string)
	Servers() []Server
}

// Server is a Server that can be stored in the collection
type Server interface {
	Host() string
	SetHost(string)
	Port() uint16
	SetPort(int)
	Scheme() string
	SetScheme(string)
	String() string
	URL() (u *url.URL, err error)
	HostPort() string
}

// NewServer creates a new server instance
func NewServer(host string, port int, scheme string) Server {
	return &BasicServer{
		host:   host,
		port:   uint16(port),
		scheme: scheme,
	}
}

// NewServers creates a new server collection
func NewServers(s ...Server) Servers {
	return &servers{
		servers: s,
	}
}

// New creates a new Cache
func New(identity string, maxAge time.Duration, resolver Resolver, log *logrus.Entry) *Cache {
	return &Cache{
		identity: identity,
		cache:    make(map[query]*entry),
		maxAge:   maxAge,
		resolver: resolver,
		log:      log,
	}
}
