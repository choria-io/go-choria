package network

import (
	"github.com/choria-io/go-srvcache"
	gnatsd "github.com/nats-io/nats-server/v2/server"
)

func (s *Server) setupLeafNodes() (err error) {
	if s.config.Choria.NetworkLeafPort == 0 {
		return nil
	}

	s.log.Infof("Starting Broker Leafnode support listening on %s:%d", s.config.Choria.NetworkListenAddress, s.config.Choria.NetworkLeafPort)

	s.opts.LeafNode.Host = s.config.Choria.NetworkListenAddress
	s.opts.LeafNode.Port = s.config.Choria.NetworkLeafPort
	s.opts.LeafNode.NoAdvertise = true

	if s.IsTLS() {
		s.opts.LeafNode.TLSConfig = s.opts.TLSConfig
		s.opts.LeafNode.TLSTimeout = s.opts.TLSTimeout
	}

	for _, r := range s.config.Choria.NetworkLeafRemotes {
		account := s.extractKeyedConfigString("leafnode_remote", r, "account", "")
		credentials := s.extractKeyedConfigString("leafnode_remote", r, "credentials", "")
		urlStr := s.extractKeyedConfigString("leafnode_remote", r, "url", "")

		if urlStr == "" {
			s.log.Errorf("Leafnode %s has no remote url, ignoring", r)
			continue
		}

		urlSrvs, err := srvcache.StringHostsToServers([]string{urlStr}, "leafnode")
		if err != nil {
			s.log.Errorf("Could not parse URL for leafnode remote %s url '%s': %s", r, urlStr, err)
			continue
		}

		if urlSrvs.Count() != 1 {
			s.log.Errorf("Could not parse URL for leafnode remote %s url '%s': need exactly 1 url", r, urlStr)
			continue
		}

		urlU, err := urlSrvs.URLs()
		if err != nil {
			s.log.Errorf("Could not parse URL for leafnode remote %s url '%s': %s", r, urlStr, err)
			continue
		}

		remote := &gnatsd.RemoteLeafOpts{LocalAccount: account, Credentials: credentials, URL: urlU[0]}

		remote.TLSTimeout = s.opts.LeafNode.TLSTimeout

		if s.IsTLS() {
			remote.TLS = true
			remote.TLSConfig = s.opts.LeafNode.TLSConfig
		}

		tlsc, disable, err := s.extractTLSCFromKeyedConfig("leafnode_remote", r)
		if err != nil {
			s.log.Errorf("Could not configure custom TLS for leafnode remote %s: %s", r, err)
			continue
		}
		if disable {
			s.log.Warnf("Disabling TLS for leafnode remote %s", r)
			remote.TLSConfig = nil
			remote.TLS = false
		} else if tlsc != nil {
			s.log.Infof("Using custom TLS config for leafnode remote %s", r)
			remote.TLSConfig = tlsc
			remote.TLS = true
		}

		s.opts.LeafNode.Remotes = append(s.opts.LeafNode.Remotes, remote)
		s.log.Infof("Added remote Leafnode %s with remote %s", r, remote.URL.String())
	}

	return nil
}
