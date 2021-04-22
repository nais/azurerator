package config

import (
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"sort"
	"strings"
	"time"
)

type Config struct {
	Azure          AzureConfig    `json:"azure"`
	MetricsAddr    string         `json:"metrics-address"`
	ClusterName    string         `json:"cluster-name"`
	Debug          bool           `json:"debug"`
	SecretRotation SecretRotation `json:"secret-rotation"`
	Validations    Validations    `json:"validations"`
}

type AzureConfig struct {
	Auth                      AzureAuth     `json:"auth"`
	Tenant                    AzureTenant   `json:"tenant"`
	PermissionGrantResourceId string        `json:"permissiongrant-resource-id"`
	Features                  AzureFeatures `json:"features"`
}

type AzureTenant struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (a AzureTenant) String() string {
	return fmt.Sprintf("%s - %s", a.Name, a.Id)
}

type AzureAuth struct {
	ClientId     string `json:"client-id"`
	ClientSecret string `json:"client-secret"`
}

type AzureFeatures struct {
	TeamsManagement           TeamsManagement           `json:"teams-management"`
	ClaimsMappingPolicies     ClaimsMappingPolicies     `json:"claims-mapping-policies"`
	GroupsAssignment          GroupsAssignment          `json:"groups-assignment"`
	AppRoleAssignmentRequired AppRoleAssignmentRequired `json:"app-role-assignment-required"`
}

type TeamsManagement struct {
	Enabled            bool   `json:"enabled"`
	ServicePrincipalId string `json:"service-principal-id"`
}

type ClaimsMappingPolicies struct {
	Enabled         bool   `json:"enabled"`
	NavIdent        string `json:"navident"`
	AzpName         string `json:"azp_name"`
	AllCustomClaims string `json:"all-custom-claims"`
}

type AppRoleAssignmentRequired struct {
	Enabled bool `json:"enabled"`
}

type GroupsAssignment struct {
	Enabled         bool   `json:"enabled"`
	AllUsersGroupId string `json:"all-users-group-id"`
}

type SecretRotation struct {
	MaxAge time.Duration `json:"max-age"`
}

type Validations struct {
	Tenant Validation `json:"tenant"`
}

type Validation struct {
	Required bool `json:"required"`
}

// Configuration options
const (
	AzureClientId                                     = "azure.auth.client-id"
	AzureClientSecret                                 = "azure.auth.client-secret"
	AzureTenantId                                     = "azure.tenant.id"
	AzureTenantName                                   = "azure.tenant.name"
	AzurePermissionGrantResourceId                    = "azure.permissiongrant-resource-id"
	AzureFeaturesClaimsMappingPoliciesEnabled         = "azure.features.claims-mapping-policies.enabled"
	AzureFeaturesClaimsMappingPoliciesNavIdent        = "azure.features.claims-mapping-policies.navident"
	AzureFeaturesClaimsMappingPoliciesAzpName         = "azure.features.claims-mapping-policies.azp_name"
	AzureFeaturesClaimsMappingPoliciesAllCustomClaims = "azure.features.claims-mapping-policies.all-custom-claims"
	AzureFeaturesTeamsManagementEnabled               = "azure.features.teams-management.enabled"
	AzureFeaturesTeamsManagementServicePrincipalId    = "azure.features.teams-management.service-principal-id"
	AzureFeaturesGroupsAssignmentEnabled              = "azure.features.groups-assignment.enabled"
	AzureFeaturesGroupsAllUsersGroupId                = "azure.features.groups-assignment.all-users-group-id"
	AzureFeaturesAppRoleAssignmentRequiredEnabled     = "azure.features.app-role-assignment-required.enabled"
	MetricsAddress                                    = "metrics-address"
	ClusterName                                       = "cluster-name"
	DebugEnabled                                      = "debug"
	ValidationsTenantRequired                         = "validations.tenant.required"
	SecretRotationMaxAge                              = "secret-rotation.max-age"
)

func init() {
	// Automatically read configuration options from environment variables.
	// e.g. --azure.client.id will be configurable using AZURERATOR_AZURE_CLIENT_ID.
	viper.SetEnvPrefix("AZURERATOR")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	// Read configuration file from working directory and/or /etc.
	// File formats supported include JSON, TOML, YAML, HCL, envfile and Java properties config files
	viper.SetConfigName("azurerator")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/azurerator")

	flag.String(AzureClientId, "", "Client ID for Azure AD authentication")
	flag.String(AzureClientSecret, "", "Client secret for Azure AD authentication")

	flag.String(AzureTenantId, "", "Tenant ID for Azure AD")
	flag.String(AzureTenantName, "", "Alias/name of tenant for Azure AD")

	flag.String(AzurePermissionGrantResourceId, "", "Object ID for Graph API permissions grant ('GraphAggregatorService' or 'Microsoft Graph' in Enterprise Applications under 'Microsoft Applications')")

	flag.Bool(AzureFeaturesClaimsMappingPoliciesEnabled, false, "Feature toggle for assigning custom claims-mapping policies to a service principal")
	flag.String(AzureFeaturesClaimsMappingPoliciesNavIdent, "", "Claims-mapping policy ID for NavIdent")
	flag.String(AzureFeaturesClaimsMappingPoliciesAzpName, "", "Claims-mapping policy ID for azp_name (authorized party name, i.e. displayName for the requesting application)")
	flag.String(AzureFeaturesClaimsMappingPoliciesAllCustomClaims, "", "Claims-mapping policy ID for all custom claims, i.e. NavIdent and azp_name")

	flag.Bool(AzureFeaturesTeamsManagementEnabled, false, "Feature toggle for assigning owners of matching teams to owners of applications")
	flag.String(AzureFeaturesTeamsManagementServicePrincipalId, "", "Service Principal ID for teams management application containing team groups")

	flag.Bool(AzureFeaturesGroupsAssignmentEnabled, false, "Feature toggle for assigning explicitly specified groups to applications")
	flag.String(AzureFeaturesGroupsAllUsersGroupId, "", "Group ID that contains all users in the tenant. Assigned to all application by default unless overridden by user in the custom resource.")

	flag.Bool(AzureFeaturesAppRoleAssignmentRequiredEnabled, false, "Feature toggle to enable 'appRoleAssignmentRequired' for service principals.")

	flag.String(MetricsAddress, ":8080", "The address the metric endpoint binds to.")
	flag.String(ClusterName, "", "The cluster in which this application should run")
	flag.Bool(DebugEnabled, false, "Debug mode toggle")
	flag.Bool(ValidationsTenantRequired, false, "If true, will only process resources that have a tenant defined in the spec")

	flag.Duration(SecretRotationMaxAge, 180*24*time.Hour, "Maximum duration since last rotation before triggering rotation on next reconciliation, regardless of secret name being changed.")
}

// Print out all configuration options except secret stuff.
func (c Config) Print(redacted []string) {
	ok := func(key string) bool {
		for _, forbiddenKey := range redacted {
			if forbiddenKey == key {
				return false
			}
		}
		return true
	}

	var keys sort.StringSlice = viper.AllKeys()

	keys.Sort()
	for _, key := range keys {
		if ok(key) {
			log.Printf("%s: %s", key, viper.GetString(key))
		} else {
			log.Printf("%s: ***REDACTED***", key)
		}
	}
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
	return nil
}

func decoderHook(dc *mapstructure.DecoderConfig) {
	dc.TagName = "json"
	dc.ErrorUnused = true
}

func New() (*Config, error) {
	var err error
	var cfg Config

	err = viper.ReadInConfig()
	if err != nil {
		if err.(viper.ConfigFileNotFoundError) != err {
			return nil, err
		}
	}

	flag.Parse()

	err = viper.BindPFlags(flag.CommandLine)
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&cfg, decoderHook)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
