// +build !cgo windows

package choria

import (
	"fmt"
)

func (fw *Framework) setupPKCS11() (err error) {
	return fmt.Errorf("pkcs11 is not supported in this build")
}
