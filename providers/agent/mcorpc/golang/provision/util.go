package provision

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"golang.org/x/crypto/curve25519"
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
	edchPrivate = &[32]byte{}
	_, err := io.ReadFull(rand.Reader, edchPrivate[:])
	if err != nil {
		return err
	}

	edchPublic = &[32]byte{}
	curve25519.ScalarBaseMult(edchPublic, edchPrivate)

	return nil
}

func edchSharedSecretLocked(public string) ([]byte, error) {
	pb, err := hex.DecodeString(public)
	if err != nil {
		return nil, err
	}

	if edchPrivate == nil {
		return nil, fmt.Errorf("private key not set")
	}

	return curve25519.X25519(edchPrivate[:], pb)
}

func decryptPrivateKey(privateKey string, edchPublic string) ([]byte, error) {
	if len(edchPublic) == 0 {
		return nil, fmt.Errorf("no EDCH Public Key")
	}

	if len(privateKey) == 0 {
		return nil, fmt.Errorf("no Private Key")
	}

	shared, err := edchSharedSecretLocked(edchPublic)
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

	decBlock, err := x509.DecryptPEMBlock(block, shared) //lint:ignore SA1019 there is no alternative
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
