package choria

import (
	context "context"
	"crypto/tls"
	"time"
)

// Certname determines the choria certname
func (fw *Framework) Certname() string {
	return fw.security.Identity()
}

// TLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (fw *Framework) TLSConfig() (tlsc *tls.Config, err error) {
	return fw.security.TLSConfig()
}

// Enroll performs the tasks needed to join the security system, like create
// a new certificate, csr etc
func (fw *Framework) Enroll(ctx context.Context, wait time.Duration, cb func(int)) error {
	return fw.security.Enroll(ctx, wait, cb)
}

// ValidateSecurity calls the security provider validation method and indicates
// if all dependencies are met for secure operation
func (fw *Framework) ValidateSecurity() (errors []string, ok bool) {
	return fw.security.Validate()
}
