package choria

import (
	"crypto/tls"
)

// Certname determines the choria certname
func (self *Framework) Certname() string {
	return self.security.Identity()
}

// TLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (self *Framework) TLSConfig() (tlsc *tls.Config, err error) {
	return self.security.TLSConfig()
}
