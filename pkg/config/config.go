package config

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/nais/liberator/pkg/conftools"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/nais/azureator/pkg/azure/client/application/groupmembershipclaim"
)

type Config struct {
	Azure          AzureConfig    `json:"azure"`
	ClusterName    string         `json:"cluster-name"`
	Controller     Controller     `json:"controller"`
	LeaderElection LeaderElection `json:"leader-election"`
	MetricsAddr    string         `json:"metrics-address"`
	ProbesAddr     string         `json:"probes-address"`
	SecretRotation SecretRotation `json:"secret-rotation"`
	Validations    Validations    `json:"validations"`
}

type AzureConfig struct {
	Auth                      AzureAuth       `json:"auth"`
	Delay                     AzureDelay      `json:"delay"`
	Features                  AzureFeatures   `json:"features"`
	Pagination                AzurePagination `json:"pagination"`
	PermissionGrantResourceId string          `json:"permissiongrant-resource-id"`
	Tenant                    AzureTenant     `json:"tenant"`
}

type AzureTenant struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (a AzureTenant) String() string {
	return fmt.Sprintf("%s - %s", a.Name, a.Id)
}

type AzureAuth struct {
	ClientId     string     `json:"client-id"`
	ClientSecret string     `json:"client-secret"`
	Google       GoogleAuth `json:"google"`
}

type AzureDelay struct {
	BetweenModifications time.Duration `json:"between-modifications"`
}

type AzurePagination struct {
	MaxPages int `json:"max-pages"`
}

type AzureFeatures struct {
	AppRoleAssignmentRequired AppRoleAssignmentRequired `json:"app-role-assignment-required"`
	ClaimsMappingPolicies     ClaimsMappingPolicies     `json:"claims-mapping-policies"`
	CleanupOrphans            CleanupOrphans            `json:"cleanup-orphans"`
	CustomSecurityAttributes  CustomSecurityAttributes  `json:"custom-security-attributes"`
	GroupsAssignment          GroupsAssignment          `json:"groups-assignment"`
	GroupMembershipClaim      GroupMembershipClaim      `json:"group-membership-claim"`
}

type AppRoleAssignmentRequired struct {
	Enabled bool `json:"enabled"`
}

type ClaimsMappingPolicies struct {
	Enabled bool   `json:"enabled"`
	ID      string `json:"id"`
}

type CleanupOrphans struct {
	Enabled bool `json:"enabled"`
}

type CustomSecurityAttributes struct {
	Enabled bool `json:"enabled"`
}

type GroupsAssignment struct {
	Enabled         bool     `json:"enabled"`
	AllUsersGroupId []string `json:"all-users-group-id"`
}

type GroupMembershipClaim struct {
	Default groupmembershipclaim.GroupMembershipClaim `json:"default"`
}

type GoogleAuth struct {
	Enabled   bool   `json:"enabled"`
	ProjectID string `json:"project-id"`
}

type Controller struct {
	ContextTimeout          time.Duration `json:"context-timeout"`
	MaxConcurrentReconciles int           `json:"max-concurrent-reconciles"`
	SweepInterval           time.Duration `json:"sweep-interval"`
}

type LeaderElection struct {
	Enabled   bool   `json:"enabled"`
	Namespace string `json:"namespace"`
}

type SecretRotation struct {
	MaxAge  time.Duration `json:"max-age"`
	Cleanup bool          `json:"cleanup"`
}

type Validations struct {
	Tenant Validation `json:"tenant"`
}

type Validation struct {
	Required bool `json:"required"`
}

// Configuration options
const (
	AzureClientId                                 = "azure.auth.client-id"
	AzureClientSecret                             = "azure.auth.client-secret"
	AzureAuthGoogleEnabled                        = "azure.auth.google.enabled"
	AzureAuthGoogleProjectID                      = "azure.auth.google.project-id"
	AzureTenantId                                 = "azure.tenant.id"
	AzureTenantName                               = "azure.tenant.name"
	AzurePermissionGrantResourceId                = "azure.permissiongrant-resource-id"
	AzureFeaturesClaimsMappingPoliciesEnabled     = "azure.features.claims-mapping-policies.enabled"
	AzureFeaturesClaimsMappingPoliciesID          = "azure.features.claims-mapping-policies.id"
	AzureFeaturesCustomSecurityAttributesEnabled  = "azure.features.custom-security-attributes.enabled"
	AzureFeaturesGroupsAssignmentEnabled          = "azure.features.groups-assignment.enabled"
	AzureFeaturesGroupsAllUsersGroupId            = "azure.features.groups-assignment.all-users-group-id"
	AzureFeaturesGroupMembershipClaimDefault      = "azure.features.group-membership-claim.default"
	AzureFeaturesAppRoleAssignmentRequiredEnabled = "azure.features.app-role-assignment-required.enabled"
	AzureFeaturesCleanupOrphansEnabled            = "azure.features.cleanup-orphans.enabled"
	AzureDelayBetweenModifications                = "azure.delay.between-modifications"
	AzurePaginationMaxPages                       = "azure.pagination.max-pages"

	ControllerContextTimeout          = "controller.context-timeout"
	ControllerMaxConcurrentReconciles = "controller.max-concurrent-reconciles"
	ControllerSweepInterval           = "controller.sweep-interval"

	LeaderElectionEnabled   = "leader-election.enabled"
	LeaderElectionNamespace = "leader-election.namespace"

	ClusterName    = "cluster-name"
	MetricsAddress = "metrics-address"
	ProbesAddress  = "probes-address"

	ValidationsTenantRequired = "validations.tenant.required"
	SecretRotationMaxAge      = "secret-rotation.max-age"
	SecretRotationCleanup     = "secret-rotation.cleanup"
)

func init() {
	conftools.Initialize("azurerator")
	viper.AddConfigPath("/etc/azurerator")

	flag.String(AzureClientId, "", "Client ID for Azure AD authentication")
	flag.String(AzureClientSecret, "", "Client secret for Azure AD authentication")
	flag.Bool(AzureAuthGoogleEnabled, false, "Use Google credentials with as federated credentials for auth.")
	flag.String(AzureAuthGoogleProjectID, "", "Google Project ID for Service Account when using federated credentials.")

	flag.String(AzureTenantId, "", "Tenant ID for Azure AD")
	flag.String(AzureTenantName, "", "Alias/name of tenant for Azure AD")

	flag.String(AzurePermissionGrantResourceId, "", "Object ID for Graph API permissions grant ('GraphAggregatorService' or 'Microsoft Graph' in Enterprise Applications under 'Microsoft Applications')")

	flag.Bool(AzureFeaturesAppRoleAssignmentRequiredEnabled, false, "Enable 'appRoleAssignmentRequired' for service principals.")
	flag.Bool(AzureFeaturesClaimsMappingPoliciesEnabled, false, "Assign custom claims-mapping policies to a service principal")
	flag.String(AzureFeaturesClaimsMappingPoliciesID, "", "Claims-mapping policy ID for custom claims mapping")
	flag.Bool(AzureFeaturesCustomSecurityAttributesEnabled, false, "Set custom security attributes on service principals (attribute set of 'Applications':'ManagedBy':'NAIS')")
	flag.Bool(AzureFeaturesGroupsAssignmentEnabled, false, "Assign groups to applications")
	flag.StringSlice(AzureFeaturesGroupsAllUsersGroupId, []string{}, "List of Group IDs that contains all users in the tenant. Assigned to all applications by default unless 'allowAllUsers' is set to false in the custom resource.")
	flag.String(AzureFeaturesGroupMembershipClaimDefault, groupmembershipclaim.ApplicationGroup, "Default group membership claim for Azure AD apps. Only affects new registrations.")

	flag.Bool(AzureFeaturesCleanupOrphansEnabled, false, "Feature toggle to enable cleanup of orphaned resources.")

	flag.Duration(AzureDelayBetweenModifications, 10*time.Second, "Delay between modification operations to the Graph API.")

	flag.Int(AzurePaginationMaxPages, 1000, "Max number of pages to fetch when fetching paginated resources from the Graph API.")

	flag.String(MetricsAddress, ":8080", "The address the metric endpoint binds to.")
	flag.String(ProbesAddress, ":8081", "The address the health probe listener binds to.")
	flag.String(ClusterName, "", "The cluster in which this application should run")
	flag.Bool(ValidationsTenantRequired, false, "If true, will only process resources that have a tenant defined in the spec")

	flag.Duration(ControllerContextTimeout, 5*time.Minute, "Context timeout for the reconciliation loop in the controller.")
	flag.Int(ControllerMaxConcurrentReconciles, 10, "Max concurrent reconciles.")
	flag.Duration(ControllerSweepInterval, 5*time.Minute, "Interval between periodic sweeps for apps with unassigned preAuthorizedApps.")

	flag.Bool(LeaderElectionEnabled, false, "Leader election toggle.")
	flag.String(LeaderElectionNamespace, "", "Leader election namespace.")

	flag.Duration(SecretRotationMaxAge, 120*24*time.Hour, "Maximum duration since last rotation before triggering rotation on next reconciliation, regardless of secret name being changed.")
	flag.Bool(SecretRotationCleanup, true, "Clean up unused credentials in Azure AD after rotation.")
}

func (c Config) Validate(required []string) error {
	present := func(key string) bool {
		for _, requiredKey := range required {
			if requiredKey == key {
				return len(viper.GetString(requiredKey)) > 0
			}
		}
		return true
	}
	var keys sort.StringSlice = viper.AllKeys()
	errs := make([]string, 0)

	keys.Sort()
	for _, key := range keys {
		if !present(key) {
			errs = append(errs, key)
		}
	}
	for _, key := range errs {
		log.Printf("required key '%s' not configured", key)
	}
	if len(errs) > 0 {
		return errors.New("missing configuration values")
	}

	if c.Azure.Features.ClaimsMappingPolicies.Enabled && len(c.Azure.Features.ClaimsMappingPolicies.ID) == 0 {
		return fmt.Errorf("'%s' cannot be empty when '%s' is true", AzureFeaturesClaimsMappingPoliciesID, AzureFeaturesClaimsMappingPoliciesEnabled)
	}

	return nil
}

func New() (*Config, error) {
	cfg := new(Config)

	if err := conftools.Load(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func DefaultConfig() (*Config, error) {
	cfg, err := New()
	if err != nil {
		return nil, err
	}

	maskedConfig := []string{
		AzureClientSecret,
	}
	for _, line := range conftools.Format(maskedConfig) {
		log.WithField("logger_name", "config").Info(line)
	}

	required := []string{
		AzureTenantId,
		AzureClientId,
		AzurePermissionGrantResourceId,
		ClusterName,
	}

	if cfg.Azure.Auth.Google.Enabled {
		required = append(required, AzureAuthGoogleProjectID)
	} else {
		required = append(required, AzureClientSecret)
	}

	err = cfg.Validate(required)
	if err != nil {
		return nil, err
	}
	return cfg, err
}
