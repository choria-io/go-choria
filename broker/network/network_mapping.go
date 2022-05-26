// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"fmt"
)

func (s *Server) setupMappings() (err error) {
	if len(s.config.Choria.NetworkMappings) == 0 {
		return nil
	}

	if s.choriaAccount == nil {
		return fmt.Errorf("choria account is not set")
	}

	for _, m := range s.config.Choria.NetworkMappings {
		source := s.extractKeyedConfigString("mapping", m, "source", "")
		dest := s.extractKeyedConfigString("mapping", m, "destination", "")

		s.log.Debugf("Attempting to add network mapping %q to %q", source, dest)
		if source == "" || dest == "" {
			s.log.Errorf("Network mapping %s need both a source and destination", m)
			continue
		}

		err = s.choriaAccount.AddMapping(source, dest)
		if err != nil {
			s.log.Errorf("Network mapping %s could not be added: %v", m, err)
		}
	}

	return nil
}
