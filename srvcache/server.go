package srvcache

import (
	"fmt"
	"net/url"
)

// BasicServer is a representation of a network server host and port, implements Server
type BasicServer struct {
	host   string
	port   uint16
	scheme string
}

// Host retrieves the host for the server
func (s *BasicServer) Host() string { return s.host }

// SetHost sets the host for the server
func (s *BasicServer) SetHost(host string) { s.host = host }

// Port retrieves the port for the server
func (s *BasicServer) Port() uint16 { return s.port }

// SetPort sets the port for the server
func (s *BasicServer) SetPort(port int) { s.port = uint16(port) }

// Scheme retrieves the url scheme
func (s *BasicServer) Scheme() string { return s.scheme }

// SetScheme sets the url scheme
func (s *BasicServer) SetScheme(scheme string) { s.scheme = scheme }

// HostPort is a string in hostname:port format
func (s *BasicServer) HostPort() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

// URL creates a correct url from the server if scheme is known
func (s *BasicServer) URL() (u *url.URL, err error) {
	if s.Scheme() == "" {
		return u, fmt.Errorf("server %s:%d has no scheme, cannot make a URL", s.host, s.port)
	}

	ustring := fmt.Sprintf("%s://%s:%d", s.scheme, s.host, s.port)

	u, err = url.Parse(ustring)
	if err != nil {
		return u, fmt.Errorf("could not parse %s: %s", ustring, err)
	}

	return u, err
}

// String is a string representation of the server in url format
func (s *BasicServer) String() string {
	return fmt.Sprintf("%s://%s:%d", s.scheme, s.host, s.port)
}
