package crypto

import (
	"crypto/rsa"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func CreateSignedJwt(credential v1alpha1.AzureAdCredential, jwkPair JwkPair) (string, error) {
	signingKey := jose.SigningKey{Algorithm: jose.RS256, Key: jwkPair.Private.Key.(*rsa.PrivateKey)}
	signingOptions := (&jose.SignerOptions{}).WithType("JWT")
	rsaSigner, err := jose.NewSigner(signingKey, signingOptions)
	if err != nil {
		return "", err
	}

	claims := struct {
		Issuer                string `json:"iss"`
		Audience              string `json:"sub"`
		CertificateThumbprint string `json:"x5t"`
	}{
		Issuer:                credential.Status.ClientId,
		Audience:              "https://graph.microsoft.com",
		CertificateThumbprint: string(jwkPair.Public.CertificateThumbprintSHA1),
	}
	token, err := jwt.Signed(rsaSigner).Claims(claims).CompactSerialize()

	if err != nil {
		return "", err
	}
	return token, nil
}
