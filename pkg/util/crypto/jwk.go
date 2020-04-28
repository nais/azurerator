package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"

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

func GenerateJwkPair(application v1alpha1.AzureAdCredential) (JwkPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return JwkPair{}, fmt.Errorf("failed to generate RSA keypair: %w", err)
	}
	return mapToJwkPair(privateKey, application)
}

func mapToJwkPair(privateKey *rsa.PrivateKey, application v1alpha1.AzureAdCredential) (JwkPair, error) {
	template := CertificateTemplate(application)
	cert, err := GenerateCertificate(template, privateKey)
	if err != nil {
		return JwkPair{}, err
	}
	certificates := []*x509.Certificate{cert}
	x5tSHA1 := sha1.Sum(certificates[0].Raw)
	x5tSHA256 := sha256.Sum256(certificates[0].Raw)
	keyId := base64.RawURLEncoding.EncodeToString(x5tSHA1[:])

	jwk := jose.JSONWebKey{
		Key:                         privateKey,
		KeyID:                       keyId,
		Use:                         KeyUseSignature,
		Certificates:                certificates,
		CertificateThumbprintSHA1:   x5tSHA1[:],
		CertificateThumbprintSHA256: x5tSHA256[:],
	}
	jwkPublic := jwk.Public()
	return JwkPair{
		Private:   jwk,
		Public:    jwkPublic,
		PublicPem: ConvertToPem(jwkPublic.Certificates[0]),
	}, nil
}
