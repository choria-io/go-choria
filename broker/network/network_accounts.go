package network

import (
	"fmt"

	"github.com/nats-io/nats-server/v2/server"
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

	if s.config.Choria.NetworkProvisioningTokenSignerFile != "" {
		s.provisioningAccount, _ = s.gnatsd.LookupOrRegisterAccount("provisioning")
		if s.provisioningAccount == nil {
			return fmt.Errorf("provisioning account creation failed")
		}

		// ensure that lifecycle events make it to choria account for observation and ingesting into streams
		err = s.provisioningAccount.AddStreamExport("choria.lifecycle.>", []*server.Account{s.choriaAccount})
		if err != nil {
			s.log.Warnf("Could not export lifecycle into Choria account")
		}

		err = s.choriaAccount.AddStreamImport(s.provisioningAccount, "choria.lifecycle.>", "")
		if err != nil {
			s.log.Warnf("Could not import lifecycle events from Provisioning account")
		}
	}

	return nil
}
