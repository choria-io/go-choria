package srvcache

import (
	"net/url"
)

// Servers is a collection of Server
type Servers struct {
	servers []Server
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

// NewServers creates a new server collection
func NewServers(servers ...Server) *Servers {
	return &Servers{
		servers: servers,
	}
}

// Servers returns a copy of the server list, changes made to server instances
// will not be reflected in the source collection
func (s *Servers) Servers() []Server {
	servers := make([]Server, len(s.servers))
	for i, srv := range s.servers {
		servers[i] = srv
	}

	return servers
}

// Strings returns a list of urls for each Server
func (s *Servers) Strings() (urls []string) {
	urls = make([]string, len(s.servers))

	for i, s := range s.servers {
		urls[i] = s.String()
	}

	return urls
}

// URLs returns a list of *url.URL for each Server
func (s *Servers) URLs() (urls []*url.URL, err error) {
	urls = make([]*url.URL, len(s.servers))

	for i, s := range s.servers {
		urls[i], err = s.URL()
		if err != nil {
			return nil, err
		}
	}

	return urls, nil
}

// HostPorts returns a list of host:port strings for each server
func (s *Servers) HostPorts() (hps []string) {
	hps = make([]string, len(s.servers))

	for i, s := range s.servers {
		hps[i] = s.HostPort()
	}

	return hps
}
