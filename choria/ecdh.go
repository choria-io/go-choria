package choria

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
)

// ECDHKeyPair create a keypair for key exchange using curve 25519
//
// This can be used to do Diffie-Hellman key exchange using Curve 25519 keys
//
// 		leftPri, leftPub, _ := ECDHKeyPair()
//
// Here we send leftPub to the remote end
//
// 		rightPri, rightPub, _ := ECDHKeyPair()
//
// Right can now figure out a shared secret:
//
// 		secret, err := ECDHSharedSecret(rightPri, leftPub)
//
// Right now does whatever needs doing with the shared
// secret and sends back rightPub to the left hand
//
// Left can now figure out the same shared secret:
//
// 		secret, err := ECDHSharedSecret(leftPri, rightPub)
//
// And decode any data encrypted using the shared secret,
// no shared keys ever traverse the network
func ECDHKeyPair() (pri []byte, pub []byte, err error) {
	private := [32]byte{}
	_, err = io.ReadFull(rand.Reader, private[:])
	if err != nil {
		return nil, nil, err
	}

	public := [32]byte{}
	curve25519.ScalarBaseMult(&public, &private)

	return private[:], public[:], nil
}

// ECDHSharedSecret calculates a shared secret based on a local private key and a remote public key
func ECDHSharedSecret(localPrivate []byte, remotePub []byte) ([]byte, error) {
	return curve25519.X25519(localPrivate, remotePub)
}

// ECDHSharedSecretString creates a shared secret in string form that can be decoded using hex.DecodeString
func ECDHSharedSecretString(localPrivate string, remotePub string) (string, error) {
	priv, err := hex.DecodeString(localPrivate)
	if err != nil {
		return "", err
	}

	pub, err := hex.DecodeString(remotePub)
	if err != nil {
		return "", err
	}

	s, err := curve25519.X25519(priv, pub)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", s), nil
}
