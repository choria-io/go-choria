package pkcs11sec

import (
	"github.com/choria-io/go-config"
	"github.com/sirupsen/logrus"
)

type Option func(*Pkcs11Security) error

func WithChoriaConfig(c *config.Config) Option {
	return func(p *Pkcs11Security) error {
		cfg := Config{
			AllowList:            c.Choria.CertnameWhitelist,
			DisableTLSVerify:     c.DisableTLSVerify,
			PrivilegedUsers:      c.Choria.PrivilegedUsers,
			CAFile:               c.Choria.FileSecurityCA,
			CertCacheDir:         c.Choria.FileSecurityCache,
			AlwaysOverwriteCache: c.Choria.SecurityAlwaysOverwriteCache,
			PKCS11DriverFile:     c.Choria.PKCS11DriverFile,
			PKCS11Slot:           uint(c.Choria.PKCS11Slot),
		}

		p.conf = &cfg

		return nil
	}
}

func WithLog(l *logrus.Entry) Option {
	return func(p *Pkcs11Security) error {
		p.log = l.WithFields(logrus.Fields{"ssl": "pkcs11"})

		return nil
	}
}

func WithPin(pin string) Option {
	return func(p *Pkcs11Security) error {
		p.pin = &pin

		return nil
	}
}
