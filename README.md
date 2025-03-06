# azurerator

Azurerator is a Kubernetes cluster operator for automated registration and lifecycle management of Azure Active
Directory applications.

This specific implementation is tailored towards managing Azure AD applications within a single tenant for use in Web
APIs, i.e. both application and user authentication and authorization.

## For [NAIS](https://nais.io) end-users

See <https://doc.nais.io/security/auth/azure-ad>

## CRD

The operator introduces a new Kind `AzureAdApplication` (shortname `azureapp`), and acts upon changes to resources of
this kind.

See the spec
in [liberator](https://github.com/nais/liberator/blob/main/config/crd/bases/nais.io_azureadapplications.yaml) for
details.

An example resource is available in [config/samples/azureadapplication.yaml](./config/samples/azureadapplication.yaml).

## Lifecycle

![overview][overview]

See [lifecycle](./docs/lifecycle.md) for details.

[overview]: ./docs/sequence.svg "Sequence diagram"

## Usage

### Installation

```shell script
make install
```

### Azure AD Setup

You will need the credentials for an Azure AD application with the following Application API permissions for Microsoft Graph:

- `Application.ReadWrite.All` or `Application.ReadWrite.Owned`
  - If you use `Application.ReadWrite.Owned`, Azurerator will only be able to manage applications and service principals that it has created.
    It will no longer be able to process these resources if removed as an owner, unless the `Application.ReadWrite.All` permission is granted.
- `DelegatedPermissionGrant.ReadWrite.All`
- `GroupMember.Read.All` (optional, only needed for the groups-assignment feature)
- `Policy.Read.All` (optional, only needed for the claims-mapping policies feature)
- `CustomSecAttributeAssignment.ReadWrite.All` (optional, only needed for the custom security attributes feature)

Finally, in order to ensure that Azurerator may pre-approve delegated API permissions for the managed applications,
you will need to find and configure the `azure.permissiongrant-resource-id` configuration flag.

This ID is the _Object ID_ of an Azure AD Enterprise Application that is unique to each tenant. 

You will find this under either the name of `GraphAggregatorService` or `Microsoft Graph`.
Look for an Enterprise Application that has an _Application ID_ equal to `00000003-0000-0000-c000-000000000000`.

### Configuration

Azurerator can be configured using either command-line flags or equivalent environment variables (i.e. `-`, `.` -> `_`
and uppercase), with `AZURERATOR_` as prefix. E.g.:

```text
azure.auth.client-id -> AZURERATOR_AZURE_AUTH_CLIENT_ID
```

The following flags are available:

```shell
--azure.auth.client-id string                                       Client ID for Azure AD authentication
--azure.auth.client-secret string                                   Client secret for Azure AD authentication
--azure.delay.between-modifications duration                        Delay between modification operations to the Graph API. (default 3s)
--azure.features.app-role-assignment-required.enabled               Enable 'appRoleAssignmentRequired' for service principals.
--azure.features.claims-mapping-policies.id string                  Claims-mapping policy ID for custom claims mapping
--azure.features.claims-mapping-policies.enabled                    Assign custom claims-mapping policies to a service principal
--azure.features.custom-security-attributes.enabled                 Set custom security attributes on service principals (attribute set of 'Applications':'ManagedBy':'NAIS')  
--azure.features.cleanup-orphans.enabled                            Enable cleanup of orphaned resources.
--azure.features.group-membership-claim.default string              Default group membership claim for Azure AD apps. Only affects new registrations. (default "ApplicationGroup")
--azure.features.groups-assignment.all-users-group-id string        Group ID that contains all users in the tenant. Assigned to all application by default unless overridden by user in the custom resource.
--azure.features.groups-assignment.enabled                          Assign groups to applications
--azure.pagination.max-pages int                                    Max number of pages to fetch when fetching paginated resources from the Graph API. (default 1000)
--azure.permissiongrant-resource-id string                          Object ID for Graph API permissions grant ('GraphAggregatorService' or 'Microsoft Graph' in Enterprise Applications under 'Microsoft Applications')
--azure.tenant.id string                                            Tenant ID for Azure AD
--azure.tenant.name string                                          Alias/name of tenant for Azure AD
--cluster-name string                                               The cluster in which this application should run
--controller.context-timeout duration                               Context timeout for the reconciliation loop in the controller. (default 5m0s)
--kafka.brokers strings                                             Comma-separated list of Kafka brokers, HOST:PORT. (default [localhost:9092])
--kafka.enabled                                                     Toggle for enabling Kafka to allow synchronization of events between Azurerator instances.
--kafka.max-processing-time duration                                Maximum processing time of Kafka messages. (default 10s)
--kafka.retry-interval duration                                     Retry interval for Kafka operations. (default 5s)
--kafka.tls.ca-path string                                          Path to Kafka TLS CA certificate.
--kafka.tls.certificate-path string                                 Path to Kafka TLS certificate.
--kafka.tls.enabled                                                 Use TLS for connecting to Kafka.
--kafka.tls.private-key-path string                                 Path to Kafka TLS private key.
--kafka.topic string                                                Name of the Kafka topic that Azurerator should use. (default "azurerator-events")
--leader-election.enabled                                           Leader election toggle.
--leader-election.namespace string                                  Leader election namespace.
--metrics-address string                                            The address the metric endpoint binds to. (default ":8080")
--secret-rotation.cleanup                                           Clean up unused credentials in Azure AD after rotation. (default true)
--secret-rotation.max-age duration                                  Maximum duration since last rotation before triggering rotation on next reconciliation, regardless of secret name being changed. (default 2880h0m0s)
--validations.tenant.required                                       If true, will only process resources that have a tenant defined in the spec
```

At minimum, the following configuration should be provided:

- `azure.auth.client-id`
- `azure.auth.client-secret`
- `azure.permissiongrant-resource-id`
- `azure.tenant.id`
- `azure.tenant.name`
- `cluster-name`

Equivalently, one can specify these properties using JSON, TOML, YAML, HCL, envfile and Java properties config files.
Azurerator looks for a file named `azurerator.<ext>` in the directories [`.`, `/etc/azurerator/`].

Example configuration in YAML:

```yaml
# ./azurerator.yaml

azure:
  auth:
    client-id: ""
    client-secret: ""
  tenant:
    id: ""
    name: "local.test" # e.g. your domain
  permissiongrant-resource-id: ""
cluster-name: minikube
```

## Development

After configuration, assuming you have a Kubernetes cluster running locally (e.g.
using [minikube](https://github.com/kubernetes/minikube)):

```shell script
ulimit -n 4096  # for controller-gen
make run # starts the controller

# in another terminal, apply an AzureAdApplication resource
make sample
```

Kubebuilder is required for running the tests. Install with `make kubebuilder`.

## Verifying the Azurerator image and its contents

The image is signed "keylessly" (is that a word?) using [Sigstore cosign](https://github.com/sigstore/cosign).
To verify its authenticity run
```
cosign verify \
--certificate-identity "https://github.com/nais/azurerator/.github/workflows/main.yml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
ghcr.io/nais/azurerator@sha256:<shasum>
```

The images are also attested with SBOMs in the [CycloneDX](https://cyclonedx.org/) format.
You can verify these by running
```
cosign verify-attestation --type cyclonedx \
--certificate-identity "https://github.com/nais/azurerator/.github/workflows/main.yml@refs/heads/master" \
--certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
ghcr.io/nais/azurerator@sha256:<shasum>
```
