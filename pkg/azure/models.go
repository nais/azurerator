package azure

import (
	"context"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/util/crypto"
)

const (
	AzureratorPrefix = "azurerator"
)

type OperationResult int

const (
	OperationResultCreated OperationResult = iota
	OperationResultUpdated
	OperationResultNotModified
)

type Client interface {
	Create(tx Transaction) (*ApplicationResult, error)
	Delete(tx Transaction) error
	Exists(tx Transaction) (*msgraph.Application, bool, error)
	Get(tx Transaction) (msgraph.Application, error)
	GetPreAuthorizedApps(tx Transaction) (*PreAuthorizedApps, error)
	GetServicePrincipal(tx Transaction) (msgraph.ServicePrincipal, error)
	AddCredentials(tx Transaction) (CredentialsSet, error)
	RotateCredentials(tx Transaction, existing CredentialsSet, inUse KeyIdsInUse) (CredentialsSet, error)
	ValidateCredentials(tx Transaction, existing CredentialsSet) (bool, error)
	Update(tx Transaction) (*ApplicationResult, error)
}

type Transaction struct {
	Ctx      context.Context
	Instance v1.AzureAdApplication
	Log      log.Entry
}

func (t Transaction) UpdateWithApplicationIDs(application msgraph.Application) Transaction {
	newInstance := t.Instance
	newInstance.Status.ClientId = *application.AppID
	newInstance.Status.ObjectId = *application.ID
	t.Instance = newInstance
	return t
}

func (t Transaction) UpdateWithServicePrincipalID(servicePrincipal msgraph.ServicePrincipal) Transaction {
	newInstance := t.Instance
	newInstance.Status.ServicePrincipalId = *servicePrincipal.ID
	t.Instance = newInstance
	return t
}

type ApplicationResult struct {
	ClientId           string            `json:"clientId"`
	ObjectId           string            `json:"objectId"`
	ServicePrincipalId string            `json:"servicePrincipalId"`
	PreAuthorizedApps  PreAuthorizedApps `json:"preAuthorizedApps"`
	Tenant             string            `json:"tenant"`
	Result             OperationResult   `json:"result"`
}

func (a ApplicationResult) IsNotModified() bool {
	return a.Result == OperationResultNotModified
}

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

// Resource contains metadata that identifies a resource (e.g. User, Group, Application, or Service Principal) within Azure AD.
type Resource struct {
	Name          string        `json:"name"`
	ClientId      string        `json:"clientId"`
	ObjectId      string        `json:"-"`
	PrincipalType PrincipalType `json:"-"`
}

// DisplayName is the display name for the Graph API Application resource
type DisplayName = string

// ClientId is the Client ID / Application ID for the Graph API Application resource
type ClientId = string

// ObjectId is the Object ID for the Graph API Application resource
type ObjectId = string

// ServicePrincipalId is the Object ID for the Graph API Service Principal resource
type ServicePrincipalId = string

// IdentifierUris is a list of unique Application ID URIs for the Graph API Application resource
type IdentifierUris = []string

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

type PrincipalType string

const (
	PrincipalTypeGroup            PrincipalType = "Group"
	PrincipalTypeServicePrincipal PrincipalType = "ServicePrincipal"
)
