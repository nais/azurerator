package azure

import (
	"context"

	"github.com/go-logr/logr"
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
	Rotate(tx Transaction, app Application) (Application, error)
	Update(tx Transaction) (Application, error)
}

type Transaction struct {
	Ctx      context.Context
	Resource v1alpha1.AzureAdApplication
	Log      logr.Logger
}

type Application struct {
	Credentials       Credentials        `json:"credentials"`
	ClientId          string             `json:"clientId"`
	ObjectId          string             `json:"objectId"`
	CertificateKeyId  string             `json:"certificateKeyId"`
	PasswordKeyId     string             `json:"passwordKeyId"`
	PreAuthorizedApps []PreAuthorizedApp `json:"preAuthorizedApps"`
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

type PreAuthorizedApp struct {
	Name     string `json:"name"`
	ClientId string `json:"clientId"`
}
