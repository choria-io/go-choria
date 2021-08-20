package provision

import (
	"bytes"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

func abort(msg string, reply *mcorpc.Reply) {
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = msg
}

func checkToken(token string, reply *mcorpc.Reply) bool {
	if build.ProvisionToken == "" {
		return true
	}

	if token != build.ProvisionToken {
		log.Errorf("Incorrect Provisioning Token %s given", token)
		abort("Incorrect provision token supplied", reply)
		return false
	}

	return true
}

func updateEDCHLocked() error {
	var err error

	edchPrivate, edchPublic, err = choria.EDCHKeyPair()

	return err
}

func edchSharedSecretLocked(provisionerPub string) ([]byte, error) {
	pb, err := hex.DecodeString(provisionerPub)
	if err != nil {
		return nil, err
	}

	if len(edchPrivate) == 0 {
		return nil, fmt.Errorf("private key not set")
	}

	return choria.EDCHSharedSecret(edchPrivate, pb)
}

func decryptPrivateKey(privateKey string, edchPublic string) ([]byte, error) {
	if len(edchPublic) == 0 {
		return nil, fmt.Errorf("no EDCH Public Key")
	}

	if len(privateKey) == 0 {
		return nil, fmt.Errorf("no Private Key")
	}

	secret, err := edchSharedSecretLocked(edchPublic)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return nil, fmt.Errorf("bad key received")
	}

	//lint:ignore SA1019 there is no alternative
	if !x509.IsEncryptedPEMBlock(block) {
		return nil, fmt.Errorf("key is not encrypted")
	}

	decBlock, err := x509.DecryptPEMBlock(block, secret) //lint:ignore SA1019 there is no alternative
	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}
	err = pem.Encode(out, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: decBlock})
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
