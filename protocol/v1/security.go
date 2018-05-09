package v1

type SecurityProvider interface {
	CallerIdentity(caller string) (string, error)
	SignString(s string) (signature []byte, err error)
	PrivilegedVerifyStringSignature(dat string, sig []byte, identity string) bool
	PublicCertTXT() ([]byte, error)
	ChecksumString(data string) []byte
	CachePublicData(data []byte, identity string) error
}
