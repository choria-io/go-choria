// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tlssetup

import (
	"crypto/tls"
)

func findCipherSuite(cipher string) *tls.CipherSuite {

	// Second element of tls.CipherSuites is a string identifying the CipherSuite
	for _, cs := range tls.CipherSuites() {
		if cs.Name == cipher {
			return cs
		}
	}

	return nil
}

// CurvePreferenceMap is a list of supported ECC Curves, optimized for performance
var CurvePreferenceMap = map[string]tls.CurveID{
	"X25519":    tls.X25519,
	"CurveP256": tls.CurveP256,
	"CurveP384": tls.CurveP384,
	"CurveP521": tls.CurveP521,
}

// DefaultCurvePreferences returns a sorted list of ECC curves,
// reordered to default to the highest level of security.  See:
// https://blog.bracebin.com/achieving-perfect-ssl-labs-score-with-go
func DefaultCurvePreferences() []tls.CurveID {
	return []tls.CurveID{
		tls.X25519, // faster than P256, arguably more secure
		tls.CurveP256,
		tls.CurveP384,
		tls.CurveP521,
	}
}

func DefaultCipherSuites() []uint16 {
	suites := make([]uint16, len(tls.CipherSuites()))
	for x, cipher := range tls.CipherSuites() {
		suites[x] = cipher.ID
	}
	return suites
}
