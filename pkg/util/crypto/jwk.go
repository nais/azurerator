package crypto

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"

	"github.com/nais/azureator/api/v1"
	"gopkg.in/square/go-jose.v2"
)

const (
	KeyUseSignature string = "sig"
)

type Jwk struct {
	Private   jose.JSONWebKey `json:"private"`
	PublicPem []byte          `json:"publicPem"`
}

func GenerateJwk(application v1.AzureAdApplication) (Jwk, error) {
	keyPair, err := NewRSAKeyPair()
	if err != nil {
		return Jwk{}, err
	}

	template := CertificateTemplate(application)
	cert, err := GenerateCertificate(template, keyPair)
	if err != nil {
		return Jwk{}, err
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
	jwkPublic := jwk.Public()
	return Jwk{
		Private:   jwk,
		PublicPem: ConvertToPem(jwkPublic.Certificates[0]),
	}, nil
}

func (j Jwk) ToPrivateJwks() jose.JSONWebKeySet {
	return jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			j.Private,
		},
	}
}
