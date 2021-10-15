// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"fmt"
	"time"
)

func (s *Server) setupWebSockets() error {
	if s.config.Choria.NetworkWebSocketPort == 0 {
		return nil
	}

	if s.config.Choria.NetworkClientTLSAnon {
		return fmt.Errorf("disabled when anonymous TLS is configured")
	}

	s.log.Infof("Starting Broker WebSocket support on port %d", s.config.Choria.NetworkWebSocketPort)
	s.opts.Websocket.Host = s.config.Choria.NetworkListenAddress
	s.opts.Websocket.Port = s.config.Choria.NetworkWebSocketPort
	s.opts.Websocket.Compression = true
	s.opts.Websocket.AuthTimeout = s.opts.AuthTimeout
	s.opts.Websocket.HandshakeTimeout = 2 * time.Second

	if s.config.Choria.NetworkWebSocketAdvertise != "" {
		s.opts.Websocket.Advertise = s.config.Choria.NetworkWebSocketAdvertise
	}

	if s.IsTLS() {
		s.opts.Websocket.TLSConfig = s.opts.TLSConfig
	}

	return nil
}
