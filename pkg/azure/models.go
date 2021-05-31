package azure

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/util/crypto"
)

type CredentialsSet struct {
	Current Credentials `json:"current"`
	Next    Credentials `json:"next"`
}

type KeyIdsInUse struct {
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

type PreAuthorizedApps struct {
	// Valid is the list of apps that either are or can be assigned to an application in Azure AD.
	Valid []Resource `json:"valid"`
	// Invalid is the list of apps that cannot be assigned to the application in Azure AD (e.g. apps that do not exist).
	Invalid []Resource `json:"invalid"`
}

type AddedKeyCredentialSet struct {
	Current AddedKeyCredential
	Next    AddedKeyCredential
}

type AddedKeyCredential struct {
	KeyCredential msgraph.KeyCredential
	Jwk           crypto.Jwk
}
