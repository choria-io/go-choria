// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"fmt"

	gnatsd "github.com/nats-io/nats-server/v2/server"
)

func (s *Server) setupCluster() (err error) {
	peers, err := s.choria.NetworkBrokerPeers()
	if err != nil {
		return fmt.Errorf("could not determine network broker peers: %s", err)
	}

	if peers.Count() == 0 {
		s.log.Infof("Skipping clustering configuration without any peers")
		return nil
	}

	if peers.Count() > 0 && s.config.Choria.NetworkPeerPort == 0 {
		s.log.Info("Defaulting Choria Broker Peer port to 5222")
		s.config.Choria.NetworkPeerPort = 5222
	}

	s.opts.Cluster.Host = s.config.Choria.NetworkListenAddress
	s.opts.Cluster.NoAdvertise = true
	s.opts.Cluster.Port = s.config.Choria.NetworkPeerPort
	s.opts.Cluster.Username = s.config.Choria.NetworkPeerUser
	s.opts.Cluster.Password = s.config.Choria.NetworkPeerPassword
	s.opts.Cluster.PoolSize = -1

	for _, p := range peers.Servers() {
		u, err := p.URL()
		if err != nil {
			return fmt.Errorf("could not parse Peer configuration: %s", err)
		}

		s.log.Infof("Adding %s as network peer", u.String())
		s.opts.Routes = append(s.opts.Routes, u)
	}

	// Remove any host/ip that points to itself in Route
	newroutes, err := gnatsd.RemoveSelfReference(s.opts.Cluster.Port, s.opts.Routes)
	if err != nil {
		s.log.Warnf("could not remove own Self from cluster configuration: %s", err)
	} else {
		s.opts.Routes = newroutes
	}

	return
}
