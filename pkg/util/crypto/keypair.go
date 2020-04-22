package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
)

type KeyPair struct {
	Private ed25519.PrivateKey `json:"private"`
	Public  ed25519.PublicKey  `json:"public"`
}

func GenerateKeyPair() (KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, fmt.Errorf("failed to generate keypair: %w", err)
	}
	return KeyPair{
		Private: privateKey,
		Public:  publicKey,
	}, nil
}
