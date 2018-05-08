package choria

import (
	"fmt"
	"net/url"
)

// Server is a representation of a network server host and port
type Server struct {
	Host   string
	Port   int
	Scheme string
}

// URL creates a correct url from the server if scheme is known
func (self *Server) URL() (u *url.URL, err error) {
	if self.Scheme == "" {
		return u, fmt.Errorf("Server %s:%d has no scheme, cannot make a URL", self.Host, self.Port)
	}

	ustring := fmt.Sprintf("%s://%s:%d", self.Scheme, self.Host, self.Port)

	u, err = url.Parse(ustring)
	if err != nil {
		return u, fmt.Errorf("Could not parse %s: %s", ustring, err)
	}

	return
}
