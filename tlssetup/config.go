package tlssetup

import (
	"crypto/tls"
	"github.com/choria-io/go-config"
)

type Config struct {
	// CipherSuites is the uint16 values from the crypto/tls library which CipherList is translated to
	CipherSuites []uint16

	// CurvePreferences is a list of curve preferences for ECC
	CurvePreferences []tls.CurveID
}

func TLSConfig(c *config.Config) (*Config) {
	cfg := &Config{}

	if c == nil {
		cfg.CipherSuites = DefaultCipherSuites()
		cfg.CurvePreferences = DefaultCurvePreferences()

		return cfg
	} else if c.Choria.CipherSuites == nil || len(c.Choria.CipherSuites) == 0 {
		cfg.CipherSuites = DefaultCipherSuites()
	} else {
		cfg.CipherSuites = []uint16{}
		for _, cipher := range c.Choria.CipherSuites {
			cfg.CipherSuites = append(cfg.CipherSuites, CipherMap[cipher])
		}
	}

	if c.Choria.ECCCurves == nil || len(c.Choria.ECCCurves) == 0 {
		cfg.CurvePreferences = DefaultCurvePreferences()
	} else {
		cfg.CurvePreferences = []tls.CurveID{}
		for _, curve := range c.Choria.ECCCurves {
			cfg.CurvePreferences = append(cfg.CurvePreferences, CurvePreferenceMap[curve])
		}
	}

	return cfg
}