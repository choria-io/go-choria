package network

import (
	"fmt"

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
		root := fmt.Sprintf("plugin.choria.network.leafnode_remote.%s", r)
		s.log.Infof("Adding leafnode remote %s via %s", r, root)

		account := s.extractKeydConfigString("leafnode_remote", r, "account", "")
		credentials := s.extractKeydConfigString("leafnode_remote", r, "credentials", "")
		urlStr := s.extractKeydConfigString("leafnode_remote", r, "url", "")

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

		if s.IsTLS() {
			remote.TLS = true
			remote.TLSConfig = s.opts.LeafNode.TLSConfig
			remote.TLSTimeout = s.opts.LeafNode.TLSTimeout
		}

		s.opts.LeafNode.Remotes = append(s.opts.LeafNode.Remotes, remote)
		s.log.Infof("Added remote Leafnode %s with remote %s", r, remote.URL.String())
	}

	return nil
}
