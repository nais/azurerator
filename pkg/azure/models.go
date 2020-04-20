package azure

import (
	"context"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

type Client interface {
	Exists(ctx context.Context, credential v1alpha1.AzureAdCredential) (bool, error)
	Create(ctx context.Context, credential v1alpha1.AzureAdCredential) (Application, error)
	Update(ctx context.Context, credential v1alpha1.AzureAdCredential) (Application, error)
	Delete(ctx context.Context, credential v1alpha1.AzureAdCredential) error
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
	ClientId string `json:"clientId"`
	Key      Key    `json:"key"`
}

type Private struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Key          Key    `json:"key"`
}

type Key struct {
	Base64 string `json:"base64"`
}
