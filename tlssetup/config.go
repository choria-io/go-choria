// Copyright (c) 2020-2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tlssetup

import (
	"crypto/tls"
	"github.com/choria-io/go-choria/config"
)

type Config struct {
	// CipherSuites is the uint16 values from the crypto/tls library which CipherList is translated to
	CipherSuites []uint16

	// CurvePreferences is a list of curve preferences for ECC
	CurvePreferences []tls.CurveID
}

func TLSConfig(c *config.Config) *Config {
	cfg := &Config{}

	if c == nil {
		cfg.CipherSuites = DefaultCipherSuites()
		cfg.CurvePreferences = DefaultCurvePreferences()

		return cfg
	} else if len(c.Choria.CipherSuites) == 0 {
		cfg.CipherSuites = DefaultCipherSuites()
	} else {
		cfg.CipherSuites = make([]uint16, 0)
		for _, cipher := range c.Choria.CipherSuites {
			cs := findCipherSuite(cipher)
			if cs != nil {
				cfg.CipherSuites = append(cfg.CipherSuites, cs.ID)
			}
		}
	}

	if len(c.Choria.ECCCurves) == 0 {
		cfg.CurvePreferences = DefaultCurvePreferences()
	} else {
		cfg.CurvePreferences = []tls.CurveID{}
		for _, curve := range c.Choria.ECCCurves {
			cfg.CurvePreferences = append(cfg.CurvePreferences, CurvePreferenceMap[curve])
		}
	}

	return cfg
}
