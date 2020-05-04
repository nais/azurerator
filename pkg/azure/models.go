package azure

import (
	"context"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"gopkg.in/square/go-jose.v2"
)

type Client interface {
	Create(ctx context.Context, credential v1alpha1.AzureAdCredential) (Application, error)
	Delete(ctx context.Context, credential v1alpha1.AzureAdCredential) error
	Exists(ctx context.Context, credential v1alpha1.AzureAdCredential) (bool, error)
	Get(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.Application, error)
	GetByName(ctx context.Context, name string) (msgraph.Application, error)
	Rotate(ctx context.Context, credential v1alpha1.AzureAdCredential) (Application, error)
	Update(ctx context.Context, credential v1alpha1.AzureAdCredential) error
}

type Application struct {
	Credentials        Credentials `json:"credentials"`
	ClientId           string      `json:"clientId"`
	ObjectId           string      `json:"objectId"`
	CertificateKeyId   string      `json:"certificateKeyId"`
	PasswordKeyId      string      `json:"passwordKeyId"`
	ServicePrincipalId string      `json:"servicePrincipalId"`
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
