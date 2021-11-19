// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
)

func Ed25519SignWithSeedFile(f string, msg []byte) ([]byte, error) {
	_, pri, err := Ed25519KeyPairFromSeedFile(f)
	if err != nil {
		return nil, err
	}

	return Ed25519Sign(pri, msg)
}

func Ed25519Sign(pk ed25519.PrivateKey, msg []byte) ([]byte, error) {
	return ed25519.Sign(pk, msg), nil
}

func Ed25519KeyPairFromSeedFile(f string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	ss, err := os.ReadFile(f)
	if err != nil {
		return nil, nil, err
	}

	seed, err := hex.DecodeString(string(ss))
	if err != nil {
		return nil, nil, err
	}

	return Ed25519KeyPairFromSeed(seed)
}

func Ed25519KeyPairFromSeed(seed []byte) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	priK := ed25519.NewKeyFromSeed(seed)
	pubK := priK.Public().(ed25519.PublicKey)
	return pubK, priK, nil
}

func Ed25519KeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

func Ed25519KeyPairToFile(f string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pubK, priK, err := Ed25519KeyPair()
	if err != nil {
		return nil, nil, err
	}

	err = os.WriteFile(f, []byte(hex.EncodeToString(priK.Seed())), 0400)
	if err != nil {
		return nil, nil, err
	}

	return pubK, priK, nil
}
