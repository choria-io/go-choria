package network

import (
	"fmt"
	"strings"

	"github.com/choria-io/go-srvcache"
	gnatsd "github.com/nats-io/nats-server/v2/server"
)

func (s *Server) setupGateways() (err error) {
	if s.config.Choria.NetworkGatewayPort == 0 {
		return nil
	}

	if s.config.Choria.NetworkGatewayName == "" {
		return fmt.Errorf("Network Gateways require a name")
	}

	if len(s.config.Choria.NetworkGatewayRemotes) == 0 {
		return fmt.Errorf("Network Gateways require at least one remote")
	}

	s.log.Infof("Starting Broker Gateway %s listening on %s:%d", s.config.Choria.NetworkGatewayName, s.config.Choria.NetworkListenAddress, s.config.Choria.NetworkGatewayPort)

	s.opts.Gateway.Host = s.config.Choria.NetworkListenAddress
	s.opts.Gateway.Port = s.config.Choria.NetworkGatewayPort
	s.opts.Gateway.Name = s.config.Choria.NetworkGatewayName
	s.opts.Gateway.RejectUnknown = true

	if s.IsTLS() {
		s.opts.Gateway.TLSConfig = s.opts.TLSConfig
		s.opts.Gateway.TLSTimeout = s.opts.TLSTimeout
	}

	for _, r := range s.config.Choria.NetworkGatewayRemotes {
		root := fmt.Sprintf("plugin.choria.network.gateway_remote.%s", r)
		s.log.Infof("Adding gateway %s via %s", r, root)

		remote := &gnatsd.RemoteGatewayOpts{Name: r}

		urlStr := s.config.Option(root+".urls", "")
		if urlStr == "" {
			s.log.Errorf("Gateway %s has no remote url, ignoring", r)
			continue
		}

		urlsStr := []string{}
		for _, u := range strings.Split(urlStr, ",") {
			urlsStr = append(urlsStr, strings.TrimSpace(u))
		}

		urlSrvs, err := srvcache.StringHostsToServers(urlsStr, "nats")
		if err != nil {
			s.log.Errorf("Could not parse URL for gateway remote %s urls '%s': %s", r, urlStr, err)
			continue
		}

		if urlSrvs.Count() == 0 {
			s.log.Errorf("Could not parse URL for gateway remote %s url '%s': needs at least 1 url", r, urlStr)
			continue
		}

		urlU, err := urlSrvs.URLs()
		if err != nil {
			s.log.Errorf("Could not parse URL for gateway remote %s url '%s': %s", r, urlStr, err)
			continue
		}

		remote.URLs = urlU

		if s.IsTLS() {
			remote.TLSConfig = s.opts.Gateway.TLSConfig
			remote.TLSTimeout = s.opts.Gateway.TLSTimeout
		}

		s.opts.Gateway.Gateways = append(s.opts.Gateway.Gateways, remote)
		s.log.Infof("Added remote Gateway %s with servers %s", r, strings.Join(urlSrvs.Strings(), ", "))
	}

	return nil
}
