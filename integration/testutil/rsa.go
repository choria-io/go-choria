// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func LoadRSAKey(key string) (pri *rsa.PrivateKey, err error) {
	kb, err := os.ReadFile(key)
	if err != nil {
		return nil, err
	}

	privPem, _ := pem.Decode(kb)
	if privPem.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("not a rsa private key")
	}

	parsedKey, err := x509.ParsePKCS1PrivateKey(privPem.Bytes)
	if err != nil {
		return nil, err
	}

	return parsedKey, nil
}

// CreateRSAKeyAndCert public.pem and private.pem in td
func CreateRSAKeyAndCert(td string) (pri *rsa.PrivateKey, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Choria.IO Testing"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 180),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}

	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	err = os.WriteFile(filepath.Join(td, "public.pem"), out.Bytes(), 0600)
	if err != nil {
		return nil, err
	}

	out.Reset()

	blk := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	pem.Encode(out, blk)

	err = os.WriteFile(filepath.Join(td, "private.pem"), out.Bytes(), 0600)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}
