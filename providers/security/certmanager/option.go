package certmanagersec

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
)

type Option func(*CertManagerSecurity) error

func WithChoriaConfig(c *config.Config) Option {
	return func(p *CertManagerSecurity) error {
		cfg := Config{
			sslDir:               c.Choria.SSLDir,
			privilegedUsers:      c.Choria.PrivilegedUsers,
			alwaysOverwriteCache: c.Choria.SecurityAlwaysOverwriteCache,
			namespace:            c.Choria.CertManagerSecurityNamespace,
			issuer:               c.Choria.CertManagerSecurityIssuer,
			replace:              c.Choria.CertManagerSecurityReplaceCSR,
			identity:             c.Identity,
		}

		if c.Choria.NetworkClientAdvertiseName != "" {
			for _, n := range strings.Split(c.Choria.NetworkClientAdvertiseName, ",") {
				n = strings.TrimSpace(n)
				if !strings.Contains(n, "://") {
					n = fmt.Sprintf("nats://%s", n)
				}

				uri, err := url.Parse(n)
				if err != nil {
					return fmt.Errorf("could not parse alternate name %q: %s", n, err)
				}

				cfg.altnames = append(cfg.altnames, strings.Split(uri.Host, ":")[0])
			}
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
