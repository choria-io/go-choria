// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"
)

// PublicCert is the parsed public certificate
func (fw *Framework) PublicCert() (*x509.Certificate, error) {
	return fw.security.PublicCert()
}

// Certname determines the choria certname
func (fw *Framework) Certname() string {
	return fw.security.Identity()
}

// TLSConfig creates a generic TLS configuration for use by NATS, HTTPS etc
func (fw *Framework) TLSConfig() (*tls.Config, error) {
	return fw.security.TLSConfig()
}

// ClientTLSConfig creates a TLS configuration for use by NATS, HTTPS, specifically configured for clients
func (fw *Framework) ClientTLSConfig() (*tls.Config, error) {
	return fw.security.ClientTLSConfig()
}

// Enroll performs the tasks needed to join the security system, like create
// a new certificate, csr etc
func (fw *Framework) Enroll(ctx context.Context, wait time.Duration, cb func(digest string, try int)) error {
	return fw.security.Enroll(ctx, wait, cb)
}

// ValidateSecurity calls the security provider validation method and indicates
// if all dependencies are met for secure operation
func (fw *Framework) ValidateSecurity() (errors []string, ok bool) {
	return fw.security.Validate()
}

// SecurityProvider is the name of the active security provider
func (fw *Framework) SecurityProvider() string {
	return fw.security.Provider()
}
