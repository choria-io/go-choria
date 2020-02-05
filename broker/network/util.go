package network

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
)

func (s *Server) extractKeyedConfigString(prefix string, key string, property string, dflt string) (result string) {
	item := "plugin.choria.network." + prefix + "." + key + "." + property
	s.log.Debugf("Looking for config item %s", item)
	return s.config.Option(item, dflt)
}

func (s *Server) extractTLSCFromKeyedConfig(prefix string, key string) (tlsc *tls.Config, disabled bool, err error) {
	cert := s.extractKeyedConfigString(prefix, key, "tls.cert", "")
	private := s.extractKeyedConfigString(prefix, key, "tls.key", "")
	ca := s.extractKeyedConfigString(prefix, key, "tls.ca", "")
	verifyS := s.extractKeyedConfigString(prefix, key, "tls.verify", "yes")
	disableS := s.extractKeyedConfigString(prefix, key, "tls.disable", "no")

	verify := !(verifyS == "false" || verifyS == "no" || verifyS == "off" || verifyS == "0")
	disabled = !(disableS == "false" || disableS == "no" || disableS == "off" || disableS == "0")

	if private == "" && cert == "" && ca == "" {
		return nil, disabled, nil
	}

	s.log.Debugf("Generating custom TLS for %s.%s: cert: '%s' private: '%s' ca: '%s' verify: %v disable: %v", prefix, key, cert, private, ca, verify, disabled)

	tlsc, err = s.genTLSc(private, cert, ca, verify)
	return tlsc, disabled, err
}

// Started determines if the server have been started
func (s *Server) Started() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.started
}

// IsTLS determines if tls should be enabled
func (s *Server) IsTLS() bool {
	return !s.config.DisableTLS
}

// IsVerifiedTLS determines if tls should be enabled
func (s *Server) IsVerifiedTLS() bool {
	return !s.config.DisableTLSVerify
}

func (s *Server) genTLSc(pri string, pub string, ca string, verify bool) (tlsc *tls.Config, err error) {
	tlsc = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if pri != "" && pub != "" {
		cert, err := tls.LoadX509KeyPair(pub, pri)
		if err != nil {
			return nil, fmt.Errorf("could not load certificate %s and key %s: %s", pub, pri, err)
		}

		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("error parsing certificate: %v", err)
		}

		tlsc.Certificates = []tls.Certificate{cert}
	}

	if ca != "" {
		caCert, err := ioutil.ReadFile(ca)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsc.ClientCAs = caCertPool
		tlsc.RootCAs = caCertPool
	}

	if !verify {
		tlsc.InsecureSkipVerify = true
	}

	tlsc.BuildNameToCertificate()

	return tlsc, nil
}

func fileExists(f string) bool {
	stat, err := os.Stat(f)
	if err != nil {
		return false
	}

	return !stat.IsDir()
}
