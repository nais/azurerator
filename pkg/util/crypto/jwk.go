package crypto

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"fmt"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"gopkg.in/square/go-jose.v2"
)

const (
	KeyUseSignature string = "sig"
)

type JwkPair struct {
	Private jose.JSONWebKey `json:"private"`
	Public  jose.JSONWebKey `json:"public"`
}

func GenerateJwks(application v1alpha1.AzureAdCredential) (JwkPair, error) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		return JwkPair{}, fmt.Errorf("failed to generate jwks: %w", err)
	}
	return keyPair.mapToJwkPair(application)
}

func (in *KeyPair) mapToJwkPair(application v1alpha1.AzureAdCredential) (JwkPair, error) {
	template := Template(application)
	cert, err := GenerateCertificate(template, in.Private)
	if err != nil {
		return JwkPair{}, err
	}
	certificates := []*x509.Certificate{cert}
	x5tSHA1 := sha1.Sum(certificates[0].Raw)
	x5tSHA256 := sha256.Sum256(certificates[0].Raw)
	return JwkPair{
		Private: jose.JSONWebKey{
			Key:                         in.Private,
			KeyID:                       uuid.New().String(),
			Use:                         KeyUseSignature,
			Certificates:                certificates,
			CertificateThumbprintSHA1:   x5tSHA1[:],
			CertificateThumbprintSHA256: x5tSHA256[:],
		},
		Public: jose.JSONWebKey{
			Key:                         in.Public,
			KeyID:                       uuid.New().String(),
			Use:                         KeyUseSignature,
			Certificates:                certificates,
			CertificateThumbprintSHA1:   x5tSHA1[:],
			CertificateThumbprintSHA256: x5tSHA256[:],
		},
	}, nil
}
