package azure

import (
	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

type Client interface {
	ApplicationExists(credential v1alpha1.AzureAdCredential) (bool, error)
	RegisterApplication(credential v1alpha1.AzureAdCredential) (Application, error)
	UpdateApplication(credential v1alpha1.AzureAdCredential) (Application, error)
	DeleteApplication(credential v1alpha1.AzureAdCredential) error
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
