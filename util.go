package network

func (s *Server) extractKeydConfigString(prefix string, key string, property string, dflt string) (result string) {
	return s.config.Option("plugin.choria.network."+prefix+"."+key+"."+property, dflt)
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
