# Configuration

Azurerator can be configured using either command-line flags or equivalent environment variables (i.e. `-`, `.` -> `_`
and uppercase), with `AZURERATOR_` as prefix. E.g.:

```text
azure.auth.client-id -> AZURERATOR_AZURE_AUTH_CLIENT_ID
```

Equivalently, one can specify these properties using JSON, TOML, YAML, HCL, envfile and Java properties config files.
Azurerator looks for a file named `azurerator.<ext>` in the directories [`.`, `/etc/azurerator/`].

## Entra ID Setup

You will need the credentials for an Entra ID application with the following Application API permissions for Microsoft Graph:

- `Application.ReadWrite.All` or `Application.ReadWrite.Owned`
  - If you use `Application.ReadWrite.Owned`, Azurerator will only be able to manage applications and service principals that it has created.
    It will no longer be able to process these resources if removed as an owner, unless the `Application.ReadWrite.All` permission is granted.
- `DelegatedPermissionGrant.ReadWrite.All`
- `GroupMember.Read.All` (optional, only needed for the groups-assignment feature)
- `Policy.Read.All` (optional, only needed for the claims-mapping policies feature)
- `CustomSecAttributeAssignment.ReadWrite.All` (optional, only needed for the custom security attributes feature)

### Permission Grant Resource ID

In order to ensure that Azurerator may pre-approve delegated API permissions for the managed applications,
you will need to find and configure the `azure.permissiongrant-resource-id` configuration flag.

This ID is the _Object ID_ of an Entra ID Enterprise Application that is unique to each tenant.

You will find this under either the name of `GraphAggregatorService` or `Microsoft Graph`.
Look for an Enterprise Application that has an _Application ID_ equal to `00000003-0000-0000-c000-000000000000`.

## Required Flags

At minimum, the following configuration must be provided:

- `azure.auth.client-id`
- `azure.permissiongrant-resource-id`
- `azure.tenant.id`
- `cluster-name`

Additionally, one of the following authentication methods must be configured:

- `azure.auth.client-secret` (default, when Google auth is not enabled)
- `azure.auth.google.project-id` (when `azure.auth.google.enabled` is `true`)

## All Flags

| Flag                                                    | Type     | Default             | Description                                                            |
|---------------------------------------------------------|----------|---------------------|------------------------------------------------------------------------|
| `--azure.auth.client-id`                                | string   |                     | Client ID for authentication                                           |
| `--azure.auth.client-secret`                            | string   |                     | Client secret for authentication                                       |
| `--azure.auth.google.enabled`                           | bool     | `false`             | Use Google credentials as federated credentials for auth               |
| `--azure.auth.google.project-id`                        | string   |                     | Google Project ID for Service Account when using federated credentials |
| `--azure.delay.between-modifications`                   | duration | `5s`                | Delay between modification operations to the Graph API                 |
| `--azure.features.app-role-assignment-required.enabled` | bool     | `false`             | Enable `appRoleAssignmentRequired` for service principals              |
| `--azure.features.claims-mapping-policies.enabled`      | bool     | `false`             | Assign custom claims-mapping policies to a service principal           |
| `--azure.features.claims-mapping-policies.id`           | string   |                     | Claims-mapping policy ID                                               |
| `--azure.features.cleanup-orphans.enabled`              | bool     | `false`             | Enable cleanup of orphaned resources                                   |
| `--azure.features.custom-security-attributes.enabled`   | bool     | `false`             | Set custom security attributes on service principals                   |
| `--azure.features.group-membership-claim.default`       | string   | `ApplicationGroup`  | Default group membership claim. Only affects new registrations         |
| `--azure.features.groups-assignment.all-users-group-id` | strings  |                     | List of Group IDs containing all users in the tenant                   |
| `--azure.features.groups-assignment.enabled`            | bool     | `false`             | Assign groups to applications                                          |
| `--azure.pagination.max-pages`                          | int      | `1000`              | Max pages to fetch from the Graph API                                  |
| `--azure.permissiongrant-resource-id`                   | string   |                     | Object ID for Graph API permissions grant                              |
| `--azure.tenant.id`                                     | string   |                     | Tenant ID                                                              |
| `--azure.tenant.name`                                   | string   |                     | Alias/name of tenant                                                   |
| `--cluster-name`                                        | string   |                     | The cluster in which this application runs                             |
| `--controller.context-timeout`                          | duration | `5m`                | Context timeout for the reconciliation loop                            |
| `--controller.max-concurrent-reconciles`                | int      | `10`                | Max concurrent reconciles                                              |
| `--controller.sweep-interval`                           | duration | `5m`                | Interval between periodic sweeps for unassigned preAuthorizedApps      |
| `--kafka.brokers`                                       | strings  | `localhost:9092`    | Comma-separated list of Kafka brokers                                  |
| `--kafka.enabled`                                       | bool     | `false`             | Enable Kafka for event synchronization between instances               |
| `--kafka.max-processing-time`                           | duration | `10s`               | Maximum processing time of Kafka messages                              |
| `--kafka.retry-interval`                                | duration | `5s`                | Retry interval for Kafka operations                                    |
| `--kafka.tls.ca-path`                                   | string   |                     | Path to Kafka TLS CA certificate                                       |
| `--kafka.tls.certificate-path`                          | string   |                     | Path to Kafka TLS certificate                                          |
| `--kafka.tls.enabled`                                   | bool     | `false`             | Use TLS for connecting to Kafka                                        |
| `--kafka.tls.private-key-path`                          | string   |                     | Path to Kafka TLS private key                                          |
| `--kafka.topic`                                         | string   | `azurerator-events` | Kafka topic name                                                       |
| `--leader-election.enabled`                             | bool     | `false`             | Leader election toggle                                                 |
| `--leader-election.namespace`                           | string   |                     | Leader election namespace                                              |
| `--metrics-address`                                     | string   | `:8080`             | Metrics endpoint bind address                                          |
| `--probes-address`                                      | string   | `:8081`             | Health probe listener bind address                                     |
| `--secret-rotation.cleanup`                             | bool     | `true`              | Clean up unused credentials after rotation                             |
| `--secret-rotation.max-age`                             | duration | `2880h`             | Max duration before triggering automatic rotation                      |
| `--validations.tenant.required`                         | bool     | `false`             | Only process resources that have a tenant defined in the spec          |

## Example Configuration (YAML)

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
