package credentials

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/util/crypto"
)

type Set struct {
	Current Credentials `json:"current"`
	Next    Credentials `json:"next"`
}

type KeyIDs struct {
	Used   KeyID `json:"used"`
	Unused KeyID `json:"unused"`
}

type KeyID struct {
	Certificate []string `json:"certificate"`
	Password    []string `json:"password"`
}

type Credentials struct {
	Certificate Certificate `json:"certificate"`
	Password    Password    `json:"password"`
}

type Certificate struct {
	KeyId string     `json:"keyId"`
	Jwk   crypto.Jwk `json:"jwk"`
}

type Password struct {
	KeyId        string `json:"keyId"`
	ClientSecret string `json:"clientSecret"`
}

type AddedKeyCredentialSet struct {
	Current AddedKeyCredential
	Next    AddedKeyCredential
}

type AddedKeyCredential struct {
	KeyCredential msgraph.KeyCredential
	Jwk           crypto.Jwk
}
