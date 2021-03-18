package network

import (
	"fmt"
)

func (s *Server) setupAccounts() (err error) {
	s.systemAccount, _ = s.gnatsd.LookupOrRegisterAccount("system")
	if s.systemAccount == nil {
		return fmt.Errorf("system account creation failed")
	}
	s.opts.SystemAccount = "system"

	s.choriaAccount, _ = s.gnatsd.LookupOrRegisterAccount("choria")
	if s.choriaAccount == nil {
		return fmt.Errorf("choria account creation failed")
	}

	return nil
}
