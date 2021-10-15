// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build cgo
// +build cgo

package choria

import (
	"fmt"
	"strings"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/security/pkcs11sec"
)

func (fw *Framework) setupPKCS11(signer inter.RequestSigner) (err error) {
	fw.security, err = pkcs11sec.New(
		pkcs11sec.WithChoriaConfig(fw.Config),
		pkcs11sec.WithLog(fw.Logger("security")),
		pkcs11sec.WithSigner(signer))
	if err != nil {
		return err
	}
	errors, ok := fw.security.Validate()
	if !ok {
		return fmt.Errorf("security setup is not valid, %d errors encountered: %s", len(errors), strings.Join(errors, ", "))
	}

	fw.Config.CacheBatchedTransports = true

	return nil
}
