package filesec

import (
	"fmt"
	"os"

	"github.com/choria-io/go-config"
	"github.com/sirupsen/logrus"
)

// Option is a function that can configure the File Security Provider
type Option func(*FileSecurity) error

// WithChoriaConfig optionally configures the File Security Provider from settings found in a typical Choria configuration
func WithChoriaConfig(c *config.Config) Option {
	cfg := Config{
		AllowList:            c.Choria.CertnameWhitelist,
		CA:                   c.Choria.FileSecurityCA,
		Cache:                c.Choria.FileSecurityCache,
		Certificate:          c.Choria.FileSecurityCertificate,
		DisableTLSVerify:     c.DisableTLSVerify,
		Key:                  c.Choria.FileSecurityKey,
		PrivilegedUsers:      c.Choria.PrivilegedUsers,
		Identity:             c.Identity,
		AlwaysOverwriteCache: c.Choria.SecurityAlwaysOverwriteCache,
	}

	if cn, ok := os.LookupEnv("MCOLLECTIVE_CERTNAME"); ok {
		c.OverrideCertname = cn
	}

	if c.OverrideCertname != "" {
		cfg.Identity = c.OverrideCertname
	} else if !(runtimeOs() == "windows" || uid() == 0) {
		if u, ok := os.LookupEnv("USER"); ok {
			cfg.Identity = fmt.Sprintf("%s.mcollective", u)
		}
	}

	return WithConfig(&cfg)
}

// WithConfig optionally configures the File Security Provider using its native configuration format
func WithConfig(c *Config) Option {
	return func(fs *FileSecurity) error {
		fs.conf = c

		return nil
	}
}

// WithLog configures a logger for the File Security Provider
func WithLog(l *logrus.Entry) Option {
	return func(fs *FileSecurity) error {
		fs.log = l.WithFields(logrus.Fields{"ssl": "file"})

		return nil
	}
}
