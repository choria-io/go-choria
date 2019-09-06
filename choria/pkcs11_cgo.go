// +build cgo

package choria

import (
"github.com/choria-io/go-security/pkcs11sec"
)

func (fw *Framework) setupPKCS11() (err error) {
	fw.security, err = pkcs11sec.New(pkcs11sec.WithChoriaConfig(fw.Config), pkcs11sec.WithLog(fw.Logger("security")))
	return err
}