package network

import (
	"crypto/tls"
	"fmt"

	gnatsd "github.com/nats-io/nats-server/v2/server"
)

func (s *Server) setupCluster() (err error) {
	s.opts.Cluster.Host = s.config.Choria.NetworkListenAddress
	s.opts.Cluster.NoAdvertise = true
	s.opts.Cluster.Port = s.config.Choria.NetworkPeerPort
	s.opts.Cluster.Username = s.config.Choria.NetworkPeerUser
	s.opts.Cluster.Password = s.config.Choria.NetworkPeerPassword

	peers, err := s.choria.NetworkBrokerPeers()
	if err != nil {
		return fmt.Errorf("could not determine network broker peers: %s", err)
	}

	for _, p := range peers.Servers() {
		u, err := p.URL()
		if err != nil {
			return fmt.Errorf("Could not parse Peer configuration: %s", err)
		}

		s.log.Infof("Adding %s as network peer", u.String())
		s.opts.Routes = append(s.opts.Routes, u)
	}

	// Remove any host/ip that points to itself in Route
	newroutes, err := gnatsd.RemoveSelfReference(s.opts.Cluster.Port, s.opts.Routes)
	if err != nil {
		return fmt.Errorf("could not remove own Self from cluster configuration: %s", err)
	}

	s.opts.Routes = newroutes

	if s.IsTLS() {
		s.opts.Cluster.TLSConfig = s.opts.TLSConfig
		s.opts.Cluster.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
		s.opts.Cluster.TLSConfig.RootCAs = s.opts.TLSConfig.ClientCAs
		s.opts.Cluster.TLSTimeout = s.opts.TLSTimeout
	}

	return
}
