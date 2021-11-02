// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

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

		// ensure leader election KV bucket functions in the provisioner account
		// the only key thats accessible by the provisioner account is `provisioner`
		// and he can only do info on the bucket, this way a rogue entity in there
		// cannot disrupt other leader elections.  This is a 2nd layer of protection
		// since nodes in there also lacks access to the `choria.streams.>` and `$KV.>`
		// prefixes for access to anything
		err = s.choriaAccount.AddServiceExportWithResponse("$JS.API.STREAM.INFO.KV_CHORIA_LEADER_ELECTION", server.Singleton, []*server.Account{s.provisioningAccount})
		if err == nil {
			err = s.provisioningAccount.AddServiceImport(s.choriaAccount, "choria.streams.STREAM.INFO.KV_CHORIA_LEADER_ELECTION", "$JS.API.STREAM.INFO.KV_CHORIA_LEADER_ELECTION")
			if err != nil {
				s.log.Warnf("Could not import KV_CHORIA_LEADER_ELECTION stream info API: %s", err)
			}
		} else {
			s.log.Warnf("Could not export KV_CHORIA_LEADER_ELECTION Info API to Provisioning: %s", err)
		}

		err = s.choriaAccount.AddServiceExportWithResponse("$KV.CHORIA_LEADER_ELECTION.provisioner", server.Singleton, []*server.Account{s.provisioningAccount})
		if err == nil {
			err = s.provisioningAccount.AddServiceImport(s.choriaAccount, "$KV.CHORIA_LEADER_ELECTION.provisioner", "$KV.CHORIA_LEADER_ELECTION.provisioner")
			if err != nil {
				s.log.Warnf("Could not import CHORIA_LEADER_ELECTION message subjects: %s", err)
			}
		} else {
			s.log.Warnf("Could not export CHORIA_LEADER_ELECTION message subjects to Provisioning: %s", err)
		}
	}

	return nil
}
