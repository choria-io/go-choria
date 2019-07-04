package network

import (
	"fmt"
	"path/filepath"

	gnatsd "github.com/nats-io/nats-server/v2/server"
)

func (s *Server) setupAccounts() (err error) {
	if s.config.Choria.NetworkAccountOperator == "" {
		return nil
	}

	s.log.Infof("Starting Broker Account services under operator %s", s.config.Choria.NetworkAccountOperator)

	operatorRoot := filepath.Join(filepath.Dir(s.config.ConfigFile), "accounts", "nats", s.config.Choria.NetworkAccountOperator)
	operatorPath := filepath.Join(operatorRoot, fmt.Sprintf("%s.jwt", s.config.Choria.NetworkAccountOperator))

	opc, err := gnatsd.ReadOperatorJWT(operatorPath)
	if err != nil {
		return fmt.Errorf("could not load operator JWT from %s: %s", operatorPath, err)
	}
	s.opts.TrustedOperators = append(s.opts.TrustedOperators, opc)

	s.as, err = newDirAccountStore(s.gnatsd, operatorRoot)
	if err != nil {
		return fmt.Errorf("could not start account store: %s", err)
	}

	s.opts.AccountResolver = s.as

	return nil
}

func (s *Server) setSystemAccount() (err error) {
	if s.config.Choria.NetworkAccountOperator == "" || s.config.Choria.NetworkSystemAccount == "" {
		return nil
	}

	s.log.Infof("Setting the Broker Systems Account to %s and enabling broker events", s.config.Choria.NetworkAccountOperator)
	err = s.gnatsd.SetSystemAccount(s.config.Choria.NetworkSystemAccount)
	if err != nil {
		return err
	}

	return nil
}
