package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

type KeyPair struct {
	Private crypto.PrivateKey
	Public  crypto.PublicKey
}

func NewRSAKeyPair() (KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return KeyPair{}, fmt.Errorf("failed to generate RSA keypair: %w", err)
	}
	return KeyPair{
		Private: privateKey,
		Public:  privateKey.Public(),
	}, nil
}
