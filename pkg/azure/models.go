package azure

import (
	"context"

	"github.com/nais/azureator/apis/v1alpha1"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"gopkg.in/square/go-jose.v2"
)

type Client interface {
	Create(tx Transaction) (Application, error)
	Delete(tx Transaction) error
	Exists(tx Transaction) (bool, error)
	Get(tx Transaction) (msgraph.Application, error)
	GetByName(ctx context.Context, name string) (msgraph.Application, error)
	Rotate(tx Transaction) (Application, error)
	Update(tx Transaction) error
}

type Transaction struct {
	Ctx      context.Context
	Resource v1alpha1.AzureAdApplication
}

type Application struct {
	Credentials      Credentials `json:"credentials"`
	ClientId         string      `json:"clientId"`
	ObjectId         string      `json:"objectId"`
	CertificateKeyId string      `json:"certificateKeyId"`
	PasswordKeyId    string      `json:"passwordKeyId"`
}

type Credentials struct {
	Public  Public  `json:"public"`
	Private Private `json:"private"`
}

type Public struct {
	ClientId string          `json:"clientId"`
	Jwk      jose.JSONWebKey `json:"jwk"`
}

type Private struct {
	ClientId     string          `json:"clientId"`
	ClientSecret string          `json:"clientSecret"`
	Jwk          jose.JSONWebKey `json:"jwk"`
}
