// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"crypto/ed25519"

	iu "github.com/choria-io/go-choria/internal/util"
)

func Ed24419Verify(pk ed25519.PublicKey, msg []byte, sig []byte) (bool, error) {
	return iu.Ed24419Verify(pk, msg, sig)
}

func Ed25519SignWithSeedFile(f string, msg []byte) ([]byte, error) {
	return iu.Ed25519SignWithSeedFile(f, msg)
}

func Ed25519Sign(pk ed25519.PrivateKey, msg []byte) ([]byte, error) {
	return iu.Ed25519Sign(pk, msg)
}

func Ed25519KeyPairFromSeedFile(f string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return iu.Ed25519KeyPairFromSeedFile(f)
}

func Ed25519KeyPairFromSeed(seed []byte) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return iu.Ed25519KeyPairFromSeed(seed)
}

func Ed25519KeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return iu.Ed25519KeyPair()
}

func Ed25519KeyPairToFile(f string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return iu.Ed25519KeyPairToFile(f)
}
