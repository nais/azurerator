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
	Kafka          KafkaConfig    `json:"kafka"`
	LeaderElection LeaderElection `json:"leader-election"`
	MetricsAddr    string         `json:"metrics-address"`
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
	ClientId     string `json:"client-id"`
	ClientSecret string `json:"client-secret"`
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
	GroupsAssignment          GroupsAssignment          `json:"groups-assignment"`
	GroupMembershipClaim      GroupMembershipClaim      `json:"group-membership-claim"`
	TeamsManagement           TeamsManagement           `json:"teams-management"`
}

type TeamsManagement struct {
	Enabled            bool   `json:"enabled"`
	ServicePrincipalId string `json:"service-principal-id"`
}

type ClaimsMappingPolicies struct {
	Enabled         bool   `json:"enabled"`
	AllCustomClaims string `json:"all-custom-claims"`
	AzpName         string `json:"azp_name"`
	NavIdent        string `json:"navident"`
}

type AppRoleAssignmentRequired struct {
	Enabled bool `json:"enabled"`
}

type CleanupOrphans struct {
	Enabled bool `json:"enabled"`
}

type GroupsAssignment struct {
	Enabled         bool     `json:"enabled"`
	AllUsersGroupId []string `json:"all-users-group-id"`
}

type GroupMembershipClaim struct {
	Default groupmembershipclaim.GroupMembershipClaim `json:"default"`
}

type Controller struct {
	ContextTimeout          time.Duration `json:"context-timeout"`
	MaxConcurrentReconciles int           `json:"max-concurrent-reconciles"`
}

type KafkaConfig struct {
	Enabled           bool          `json:"enabled"`
	Brokers           []string      `json:"brokers"`
	Topic             string        `json:"topic"`
	TLS               KafkaTLS      `json:"tls"`
	RetryInterval     time.Duration `json:"retry-interval"`
	MaxProcessingTime time.Duration `json:"max-processing-time"`
}

type KafkaTLS struct {
	Enabled         bool   `json:"enabled"`
	CAPath          string `json:"ca-path"`
	CertificatePath string `json:"certificate-path"`
	PrivateKeyPath  string `json:"private-key-path"`
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
	AzureFeaturesGroupMembershipClaimDefault          = "azure.features.group-membership-claim.default"
	AzureFeaturesAppRoleAssignmentRequiredEnabled     = "azure.features.app-role-assignment-required.enabled"
	AzureFeaturesCleanupOrphansEnabled                = "azure.features.cleanup-orphans.enabled"
	AzureDelayBetweenModifications                    = "azure.delay.between-modifications"
	AzurePaginationMaxPages                           = "azure.pagination.max-pages"

	ControllerContextTimeout          = "controller.context-timeout"
	ControllerMaxConcurrentReconciles = "controller.max-concurrent-reconciles"

	KafkaEnabled           = "kafka.enabled"
	KafkaBrokers           = "kafka.brokers"
	KafkaTopic             = "kafka.topic"
	KafkaRetryInterval     = "kafka.retry-interval"
	KafkaMaxProcessingTime = "kafka.max-processing-time"

	KafkaTLSEnabled         = "kafka.tls.enabled"
	KafkaTLSCAPath          = "kafka.tls.ca-path"
	KafkaTLSCertificatePath = "kafka.tls.certificate-path"
	KafkaTLSPrivateKeyPath  = "kafka.tls.private-key-path"

	LeaderElectionEnabled   = "leader-election.enabled"
	LeaderElectionNamespace = "leader-election.namespace"

	ClusterName    = "cluster-name"
	MetricsAddress = "metrics-address"

	ValidationsTenantRequired = "validations.tenant.required"
	SecretRotationMaxAge      = "secret-rotation.max-age"
	SecretRotationCleanup     = "secret-rotation.cleanup"
)

func bindNAIS() {
	viper.BindEnv(KafkaBrokers, "KAFKA_BROKERS")
	viper.BindEnv(KafkaTLSCAPath, "KAFKA_CA_PATH")
	viper.BindEnv(KafkaTLSCertificatePath, "KAFKA_CERTIFICATE_PATH")
	viper.BindEnv(KafkaTLSPrivateKeyPath, "KAFKA_PRIVATE_KEY_PATH")
}

func init() {
	conftools.Initialize("AZURERATOR")

	// Ensure NAIS Kafka variables are used
	bindNAIS()

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
	flag.StringSlice(AzureFeaturesGroupsAllUsersGroupId, []string{}, "List of Group IDs that contains all users in the tenant. Assigned to all applications by default unless 'allowAllUsers' is set to false in the custom resource.")

	flag.String(AzureFeaturesGroupMembershipClaimDefault, string(groupmembershipclaim.GroupMembershipClaimApplicationGroup), "Default group membership claim for Azure AD apps.")

	flag.Bool(AzureFeaturesAppRoleAssignmentRequiredEnabled, false, "Feature toggle to enable 'appRoleAssignmentRequired' for service principals.")

	flag.Bool(AzureFeaturesCleanupOrphansEnabled, false, "Feature toggle to enable cleanup of orphaned resources.")

	flag.Duration(AzureDelayBetweenModifications, 5*time.Second, "Delay between modification operations to the Graph API.")

	flag.Int(AzurePaginationMaxPages, 1000, "Max number of pages to fetch when fetching paginated resources from the Graph API.")

	flag.String(MetricsAddress, ":8080", "The address the metric endpoint binds to.")
	flag.String(ClusterName, "", "The cluster in which this application should run")
	flag.Bool(ValidationsTenantRequired, false, "If true, will only process resources that have a tenant defined in the spec")

	flag.Duration(ControllerContextTimeout, 5*time.Minute, "Context timeout for the reconciliation loop in the controller.")
	flag.Int(ControllerMaxConcurrentReconciles, 10, "Max concurrent reconciles.")

	flag.Bool(KafkaEnabled, false, "Toggle for enabling Kafka to allow synchronization of events between Azurerator instances.")
	flag.String(KafkaTopic, "azurerator-events", "Name of the Kafka topic that Azurerator should use.")
	flag.StringSlice(KafkaBrokers, []string{"localhost:9092"}, "Comma-separated list of Kafka brokers, HOST:PORT.")
	flag.Duration(KafkaRetryInterval, 5*time.Second, "Retry interval for Kafka operations.")
	flag.Duration(KafkaMaxProcessingTime, 10*time.Second, "Maximum processing time of Kafka messages.")
	flag.Bool(KafkaTLSEnabled, false, "Use TLS for connecting to Kafka.")
	flag.String(KafkaTLSCAPath, "", "Path to Kafka TLS CA certificate.")
	flag.String(KafkaTLSCertificatePath, "", "Path to Kafka TLS certificate.")
	flag.String(KafkaTLSPrivateKeyPath, "", "Path to Kafka TLS private key.")

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
		log.WithField("logger", "config").Info(line)
	}

	err = cfg.Validate([]string{
		AzureTenantId,
		AzureClientId,
		AzureClientSecret,
		AzurePermissionGrantResourceId,
		ClusterName,
	})
	if err != nil {
		return nil, err
	}
	return cfg, err
}
