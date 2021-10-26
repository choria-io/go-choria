// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package certmanagersec

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
)

type Option func(*CertManagerSecurity) error

func WithChoriaConfig(c *config.Config) Option {
	return func(p *CertManagerSecurity) error {
		cfg := Config{
			apiVersion:           c.Choria.CertManagerAPIVersion,
			sslDir:               c.Choria.SSLDir,
			privilegedUsers:      c.Choria.PrivilegedUsers,
			alwaysOverwriteCache: c.Choria.SecurityAlwaysOverwriteCache,
			namespace:            c.Choria.CertManagerSecurityNamespace,
			issuer:               c.Choria.CertManagerSecurityIssuer,
			replace:              c.Choria.CertManagerSecurityReplaceCSR,
			altnames:             c.Choria.CertManagerSecurityAltNames,
			identity:             c.Identity,
			legacyCerts:          c.Choria.SecurityAllowLegacyCerts,
		}

		if c.OverrideCertname == "" {
			if cn, ok := os.LookupEnv("MCOLLECTIVE_CERTNAME"); ok {
				c.OverrideCertname = cn
			}
		}

		if c.OverrideCertname != "" {
			cfg.identity = c.OverrideCertname
		}

		if cfg.sslDir == "" {
			return fmt.Errorf("plugin.choria.ssldir is required")
		}

		if cfg.identity == "" {
			return fmt.Errorf("identity could not be established")
		}

		if cfg.apiVersion == "" {
			cfg.apiVersion = "v1"
		}

		p.conf = &cfg

		return nil
	}
}

func WithLog(l *logrus.Entry) Option {
	return func(p *CertManagerSecurity) error {
		p.log = l.WithFields(logrus.Fields{"ssl": "certmanager"})

		return nil
	}
}

func WithContext(ctx context.Context) Option {
	return func(p *CertManagerSecurity) error {
		p.ctx = ctx

		return nil
	}
}
