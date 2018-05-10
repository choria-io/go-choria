package choria

import (
	context "context"
	"crypto/tls"
	"time"
)

// Certname determines the choria certname
func (self *Framework) Certname() string {
	return self.security.Identity()
}

// TLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (self *Framework) TLSConfig() (tlsc *tls.Config, err error) {
	return self.security.TLSConfig()
}

// Enroll performs the tasks needed to join the security system, like create
// a new certificate, csr etc
func (self *Framework) Enroll(ctx context.Context, wait time.Duration, cb func(int)) error {
	return self.security.Enroll(ctx, wait, cb)
}
