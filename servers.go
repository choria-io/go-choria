package srvcache

import (
	"net/url"
)

type servers struct {
	servers []Server
}

// Servers returns a copy of the server list, changes made to server instances
// will not be reflected in the source collection
func (s *servers) Servers() []Server {
	servers := make([]Server, len(s.servers))
	for i, srv := range s.servers {
		servers[i] = srv
	}

	return servers
}

// Count is the amount of servers stored
func (s *servers) Count() int {
	return len(s.servers)
}

// Strings returns a list of urls for each Server
func (s *servers) Strings() (urls []string) {
	urls = make([]string, len(s.servers))

	for i, s := range s.servers {
		urls[i] = s.String()
	}

	return urls
}

// URLs returns a list of *url.URL for each Server
func (s *servers) URLs() (urls []*url.URL, err error) {
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
func (s *servers) HostPorts() (hps []string) {
	hps = make([]string, len(s.servers))

	for i, s := range s.servers {
		hps[i] = s.HostPort()
	}

	return hps
}
