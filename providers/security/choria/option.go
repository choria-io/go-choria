// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/tlssetup"
	"github.com/sirupsen/logrus"
)

// Option is a function that can configure the Security Provider
type Option func(*ChoriaSecurity) error

// BuildInfoProvider provides info about the build
type BuildInfoProvider interface {
	ClientIdentitySuffix() string
}

// WithChoriaConfig optionally configures the Security Provider from settings found in a typical Choria configuration
func WithChoriaConfig(c *config.Config) Option {
	return func(s *ChoriaSecurity) error {
		if c.Choria.ServerTokenFile != "" {
			return fmt.Errorf("plugin.choria.security.server.token_file can not be set when the choria security provider is used")
		}

		if c.Choria.ServerTokenSeedFile != "" {
			return fmt.Errorf("plugin.choria.security.server.seed_file can not be set when the choria security provider is used")
		}

		if c.Choria.ServerAnonTLS {
			return fmt.Errorf("plugin.security.server_anon_tls can not be set when the choria security provider is used")
		}

		if c.Choria.ClientAnonTLS {
			return fmt.Errorf("plugin.security.client_anon_tls can not be set when the choria security provider is used")
		}

		if c.Choria.RemoteSignerTokenSeedFile != "" {
			return fmt.Errorf("plugin.choria.security.request_signer.seed_file can not be used when the choria security provider is used")
		}

		if c.Choria.RemoteSignerTokenFile != "" {
			return fmt.Errorf("plugin.choria.security.request_signer.token_file can not be used when the choria security provider is used")
		}

		cfg := Config{
			TLSConfig:         tlssetup.TLSConfig(c),
			RemoteSignerURL:   c.Choria.RemoteSignerURL,
			SeedFile:          filepath.FromSlash(c.Choria.ChoriaSecuritySeedFile),
			TokenFile:         filepath.FromSlash(c.Choria.ChoriaSecurityTokenFile),
			CA:                filepath.FromSlash(c.Choria.ChoriaSecurityCA),
			Certificate:       filepath.FromSlash(c.Choria.ChoriaSecurityCertificate),
			Key:               filepath.FromSlash(c.Choria.ChoriaSecurityKey),
			DisableTLSVerify:  c.DisableTLSVerify,
			InitiatedByServer: c.InitiatedByServer,
			SignedReplies:     c.Choria.ChoriaSecuritySignReplies,
			Issuers:           make(map[string]ed25519.PublicKey),
		}

		for _, signer := range c.Choria.ChoriaSecurityTrustedSigners {
			pk, err := hex.DecodeString(signer)
			if err != nil {
				return fmt.Errorf("invalid ed25519 public key: %v: %v", signer, err)
			}
			if len(pk) != ed25519.PublicKeySize {
				return fmt.Errorf("invalid ed25519 public key size: %v: %v", signer, len(pk))
			}

			cfg.TrustedTokenSigners = append(cfg.TrustedTokenSigners, pk)
		}

		for _, issuer := range c.Choria.IssuerNames {
			name := fmt.Sprintf("plugin.security.issuer.%s.public", issuer)
			pks := c.Option(name, "")
			if pks == "" {
				return fmt.Errorf("could not find option %s while adding issuer %s", name, issuer)
			}

			pk, err := hex.DecodeString(pks)
			if err != nil {
				return fmt.Errorf("invalid ed25519 public key in %s: %v", name, err)
			}

			cfg.Issuers[issuer] = pk
		}

		if c.InitiatedByServer {
			cfg.Identity = c.Identity
		} else {
			userEnvVar := "USER"
			if runtime.GOOS == "windows" {
				userEnvVar = "USERNAME"
			}

			u, ok := os.LookupEnv(userEnvVar)
			if !ok {
				return fmt.Errorf("could not determine client identity, ensure %s environment variable is set", userEnvVar)
			}

			cfg.Identity = u
		}

		s.conf = &cfg

		return nil
	}
}

// WithTokenFile sets the path to the JWT token stored in a file
func WithTokenFile(f string) Option {
	return func(s *ChoriaSecurity) error {
		s.conf.TokenFile = f
		return nil
	}
}

// WithSeedFile sets the path to the ed25519 seed stored in a file
func WithSeedFile(f string) Option {
	return func(s *ChoriaSecurity) error {
		s.conf.SeedFile = f
		return nil
	}
}

// WithSigner configures a remote request signer
func WithSigner(signer inter.RequestSigner) Option {
	return func(s *ChoriaSecurity) error {
		s.conf.RemoteSigner = signer

		return nil
	}
}

// WithConfig optionally configures the Security Provider using its native configuration format
func WithConfig(c *Config) Option {
	return func(s *ChoriaSecurity) error {
		s.conf = c

		if s.conf.TLSConfig == nil {
			s.conf.TLSConfig = tlssetup.TLSConfig(nil)
		}

		return nil
	}
}

// WithLog configures a logger for the Security Provider
func WithLog(l *logrus.Entry) Option {
	return func(s *ChoriaSecurity) error {
		s.log = l.WithFields(logrus.Fields{"security": "choria"})

		if s.conf.TLSConfig == nil {
			s.conf.TLSConfig = tlssetup.TLSConfig(nil)
		}

		return nil
	}
}
