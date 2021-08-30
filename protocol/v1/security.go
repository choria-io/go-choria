package v1

import (
	"context"
)

type SecurityProvider interface {
	CallerIdentity(caller string) (string, error)
	SignString(s string) (signature []byte, err error)
	PrivilegedVerifyStringSignature(dat string, sig []byte, identity string) bool
	PublicCertTXT() ([]byte, error)
	ChecksumString(data string) []byte
	CachePublicData(data []byte, identity string) error
	RemoteSignRequest(ctx context.Context, str []byte) (signed []byte, err error)
}
