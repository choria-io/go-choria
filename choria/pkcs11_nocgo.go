// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build !cgo
// +build !cgo

package choria

import (
	"fmt"

	"github.com/choria-io/go-choria/inter"
)

func (fw *Framework) setupPKCS11(_ inter.RequestSigner) (err error) {
	return fmt.Errorf("pkcs11 is not supported in this build")
}
