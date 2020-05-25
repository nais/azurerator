package azure

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/nais/azureator/api/v1alpha1"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"gopkg.in/square/go-jose.v2"
)

type Client interface {
	Create(tx Transaction) (Application, error)
	Delete(tx Transaction) error
	Exists(tx Transaction) (bool, error)
	Get(tx Transaction) (msgraph.Application, error)
	GetByName(ctx context.Context, name string) (msgraph.Application, error)
	GetServicePrincipal(tx Transaction) (msgraphbeta.ServicePrincipal, error)
	Rotate(tx Transaction, app Application) (Application, error)
	Update(tx Transaction) (Application, error)
}

type Transaction struct {
	Ctx      context.Context
	Instance v1alpha1.AzureAdApplication
	Log      logr.Logger
}

type Application struct {
	Credentials        Credentials        `json:"credentials"`
	ClientId           string             `json:"clientId"`
	ObjectId           string             `json:"objectId"`
	ServicePrincipalId string             `json:"servicePrincipalId"`
	CertificateKeyId   string             `json:"certificateKeyId"`
	PasswordKeyId      string             `json:"passwordKeyId"`
	PreAuthorizedApps  []PreAuthorizedApp `json:"preAuthorizedApps"`
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
	ClientSecret string          `json:"clientSecret"`
	Jwk          jose.JSONWebKey `json:"jwk"`
}

type PreAuthorizedApp struct {
	Name     string `json:"name"`
	ClientId string `json:"clientId"`
}

// DisplayName is the display name for the Graph API Application resource
type DisplayName = string

// ClientId is the Client ID / Application ID for the Graph API Application resource
type ClientId = string

// ObjectId is the Object ID for the Graph API Application resource
type ObjectId = string

// ServicePrincipalId is the Object ID for the Graph API Service Principal resource
type ServicePrincipalId = string

// IdentifierUri is the unique Application ID URI for the Graph API Application resource
type IdentifierUri = string

// Filter is the Graph API OData query option for filtering results of a collection
type Filter = string
