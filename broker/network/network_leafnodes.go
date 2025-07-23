// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"fmt"
	"strings"
	"time"

	gnatsd "github.com/nats-io/nats-server/v2/server"

	"github.com/choria-io/go-choria/srvcache"
)

func (s *Server) setupLeafNodes() (err error) {
	if s.config.Choria.NetworkLeafPort == 0 && len(s.config.Choria.NetworkLeafRemotes) == 0 {
		return nil
	}

	if s.config.Choria.NetworkLeafPort == 0 {
		s.log.Infof("Starting Broker Leafnode support with %d remote connection(s)", len(s.config.Choria.NetworkLeafRemotes))
		s.opts.LeafNode.Port = 0
	} else {
		s.log.Infof("Starting Broker Leafnode support listening on %s:%d", s.config.Choria.NetworkListenAddress, s.config.Choria.NetworkLeafPort)
	}

	for _, r := range s.config.Choria.NetworkLeafRemotes {
		account := s.extractKeyedConfigString("leafnode_remote", r, "account", "choria")
		credentials := s.extractKeyedConfigString("leafnode_remote", r, "credentials", "")
		urlStr := s.extractKeyedConfigString("leafnode_remote", r, "url", "")

		urlsStr := []string{}
		for _, u := range strings.Split(urlStr, ",") {
			urlsStr = append(urlsStr, strings.TrimSpace(u))
		}

		if urlStr == "" {
			s.log.Errorf("Leafnode %s has no remote url, ignoring", r)
			continue
		}

		urlSrvs, err := srvcache.StringHostsToServers(urlsStr, "leafnode")
		if err != nil {
			s.log.Errorf("Could not parse URL for leafnode remote %s url '%s': %s", r, urlStr, err)
			continue
		}

		if urlSrvs.Count() == 0 {
			s.log.Errorf("Could not parse URL for leafnode remote %s url '%s': needs at least 1 url", r, urlStr)
			continue
		}

		urlU, err := urlSrvs.URLs()
		if err != nil {
			s.log.Errorf("Could not parse URL for leafnode remote %s url '%s': %s", r, urlStr, err)
			continue
		}

		remote := &gnatsd.RemoteLeafOpts{
			LocalAccount:     account,
			Credentials:      credentials,
			URLs:             urlU,
			FirstInfoTimeout: 2 * time.Second,
		}

		if s.IsTLS() {
			remote.TLS = true
			remote.TLSConfig = s.opts.TLSConfig
		} else {
			s.log.Warnf("Skipping TLS setup for leafnode remote %s url %s", r, urlsStr)
		}

		tlsc, disable, err := s.extractTLSCFromKeyedConfig("leafnode_remote", r)
		if err != nil {
			s.log.Errorf("Could not configure custom TLS for leafnode remote %s: %s", r, err)
			continue
		}

		switch {
		case disable:
			s.log.Warnf("Disabling TLS for leafnode remote %s", r)
			remote.TLSConfig = nil
			remote.TLS = false

		case tlsc != nil:
			s.log.Infof("Using custom TLS config for leafnode remote %s", r)
			remote.TLSConfig = tlsc
			remote.TLS = true
		}

		s.opts.LeafNode.Remotes = append(s.opts.LeafNode.Remotes, remote)
		s.log.Infof("Added remote Leafnode %s with remote %v", r, remote.URLs)
	}

	if s.config.Choria.NetworkLeafPort > 0 {
		if s.IsTLS() {
			s.opts.LeafNode.TLSConfig = s.opts.TLSConfig
			s.opts.LeafNode.TLSTimeout = s.opts.TLSTimeout

			if s.opts.LeafNode.TLSConfig == nil {
				return fmt.Errorf("leafnode TLS is not configured")
			}

			if s.opts.LeafNode.TLSConfig.InsecureSkipVerify || !s.opts.TLSVerify {
				s.log.Warnf("Leafnode connections on port %d are not verifying TLS connections", s.opts.LeafNode.Port)
			}
		} else {
			s.log.Warnf("Skipping TLS setup for leafnode connection on port %d", s.opts.LeafNode.Port)
		}

		s.opts.LeafNode.Host = s.config.Choria.NetworkListenAddress
		s.opts.LeafNode.Port = s.config.Choria.NetworkLeafPort
		s.opts.LeafNode.NoAdvertise = true
		s.opts.LeafNode.AuthTimeout = 10.0

		advertise := s.config.Choria.NetworkClientAdvertiseName
		parts := strings.Split(s.config.Choria.NetworkClientAdvertiseName, ":")
		if len(parts) > 1 {
			advertise = fmt.Sprintf("%s:%d", parts[0], s.opts.LeafNode.Port)
		}
		s.opts.LeafNode.Advertise = advertise
	}

	return nil
}
