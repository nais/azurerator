package azure

import (
	"context"

	"github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/util/crypto"
	log "github.com/sirupsen/logrus"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

const AzureratorPrefix = "azurerator"

type Client interface {
	Create(tx Transaction) (*Application, error)
	Delete(tx Transaction) error
	Exists(tx Transaction) (bool, error)
	Get(tx Transaction) (msgraph.Application, error)
	GetServicePrincipal(tx Transaction) (msgraphbeta.ServicePrincipal, error)
	Rotate(tx Transaction, app Application) (*Application, error)
	Update(tx Transaction) (*Application, error)
}

type Transaction struct {
	Ctx      context.Context
	Instance v1.AzureAdApplication
	Log      log.Entry
}

type Application struct {
	Certificate        Certificate        `json:"certificate"`
	Password           Password           `json:"password"`
	ClientId           string             `json:"clientId"`
	ObjectId           string             `json:"objectId"`
	ServicePrincipalId string             `json:"servicePrincipalId"`
	PreAuthorizedApps  []PreAuthorizedApp `json:"preAuthorizedApps"`
	Tenant             string             `json:"tenant"`
}

type Certificate struct {
	KeyId KeyId      `json:"keyId"`
	Jwk   crypto.Jwk `json:"jwks"`
}

type Password struct {
	KeyId        KeyId  `json:"keyId"`
	ClientSecret string `json:"clientSecret"`
}

type KeyId struct {
	Latest   string   `json:"latest"`
	AllInUse []string `json:"allInUse"`
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

// GroupMembershipClaim is the type of groups to emit for tokens returned to the Application from Azure AD
type GroupMembershipClaim = string

const (
	// Emits _all_ security groups the user is a member of in the groups claim.
	GroupMembershipClaimSecurityGroup GroupMembershipClaim = "SecurityGroup"
	// Emits only the groups that are explicitly assigned to the application and the user is a member of.
	GroupMembershipClaimApplicationGroup GroupMembershipClaim = "ApplicationGroup"
	// No groups are returned.
	GroupMembershipClaimNone GroupMembershipClaim = "None"
)
