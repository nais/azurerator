package crypto

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"gopkg.in/square/go-jose.v2"
)

const (
	KeyUseSignature string = "sig"
)

type JwkPair struct {
	Private   jose.JSONWebKey `json:"private"`
	Public    jose.JSONWebKey `json:"public"`
	PublicPem []byte          `json:"publicPem"`
}

func GenerateJwkPair(application v1alpha1.AzureAdApplication) (JwkPair, error) {
	keyPair, err := NewRSAKeyPair()
	if err != nil {
		return JwkPair{}, err
	}
	return mapToJwkPair(keyPair, application)
}

func JwkToJwkPair(jwk jose.JSONWebKey) JwkPair {
	jwkPublic := jwk.Public()
	return JwkPair{
		Private:   jwk,
		Public:    jwkPublic,
		PublicPem: ConvertToPem(jwkPublic.Certificates[0]),
	}
}

func mapToJwkPair(keyPair KeyPair, application v1alpha1.AzureAdApplication) (JwkPair, error) {
	template := CertificateTemplate(application)
	cert, err := GenerateCertificate(template, keyPair)
	if err != nil {
		return JwkPair{}, err
	}
	certificates := []*x509.Certificate{cert}
	x5tSHA1 := sha1.Sum(certificates[0].Raw)
	x5tSHA256 := sha256.Sum256(certificates[0].Raw)
	keyId := base64.RawURLEncoding.EncodeToString(x5tSHA1[:])

	jwk := jose.JSONWebKey{
		Key:                         keyPair.Private,
		KeyID:                       keyId,
		Use:                         KeyUseSignature,
		Certificates:                certificates,
		CertificateThumbprintSHA1:   x5tSHA1[:],
		CertificateThumbprintSHA256: x5tSHA256[:],
	}
	return JwkToJwkPair(jwk), nil
}
