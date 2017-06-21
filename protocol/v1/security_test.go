package v1

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestSecureReply(t *testing.T) {
	request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
	request.SetMessage(`{"test":1}`)

	reply, err := NewReply(request)
	assert.Nil(t, err)
	rj, err := reply.JSON()
	assert.Nil(t, err)

	sha := sha256.Sum256([]byte(rj))

	sreply, _ := NewSecureReply(reply)
	sj, err := sreply.JSON()
	assert.Nil(t, err)

	assert.Equal(t, "choria:secure:reply:1", gjson.Get(sj, "protocol").String())
	assert.Equal(t, rj, gjson.Get(sj, "message").String())
	assert.Equal(t, base64.StdEncoding.EncodeToString(sha[:]), gjson.Get(sj, "hash").String())
	assert.True(t, sreply.Valid())
}

func TestSecureRequest(t *testing.T) {
	r, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
	r.SetMessage(`{"test":1}`)
	rj, err := r.JSON()
	assert.Nil(t, err)

	sr, _ := NewSecureRequest(r, "testdata/ssl/certs/rip.mcollective.pem", "testdata/ssl/private_keys/rip.mcollective.pem")
	sj, err := sr.JSON()
	assert.Nil(t, err)

	pubf, _ := readFile("testdata/ssl/certs/rip.mcollective.pem")
	privf, _ := readFile("testdata/ssl/private_keys/rip.mcollective.pem")

	// what signString() is doing lets just verify it
	pem, _ := pem.Decode(privf)
	pk, err := x509.ParsePKCS1PrivateKey(pem.Bytes)
	assert.Nil(t, err)
	rng := rand.Reader
	hashed := sha256.Sum256([]byte(rj))
	signature, _ := rsa.SignPKCS1v15(rng, pk, crypto.SHA256, hashed[:])

	assert.Equal(t, "choria:secure:request:1", gjson.Get(sj, "protocol").String())
	assert.Equal(t, rj, gjson.Get(sj, "message").String())
	assert.Equal(t, string(pubf), gjson.Get(sj, "pubcert").String())
	assert.Equal(t, base64.StdEncoding.EncodeToString(signature), gjson.Get(sj, "signature").String())
}
