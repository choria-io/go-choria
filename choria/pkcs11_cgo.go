// +build cgo

package choria

import (
	"fmt"
	"github.com/choria-io/go-security/pkcs11sec"
	"strings"
)

func (fw *Framework) setupPKCS11() (err error) {
	fw.security, err = pkcs11sec.New(pkcs11sec.WithChoriaConfig(fw.Config), pkcs11sec.WithLog(fw.Logger("security")))
	if err != nil {
		return err
	}
	errors, ok := fw.security.Validate()
	if !ok {
		return fmt.Errorf("security setup is not valid, %d errors encountered: %s", len(errors), strings.Join(errors, ", "))
	}
	return nil
}