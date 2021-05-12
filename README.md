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

## Development

### Installation

```shell script
kubectl apply -f <path to CRD from liberator>
```

### Azure AD Setup

You will need the credentials for an Azure AD application with the following built-in roles:

- `Application administrator`
- `Cloud application administrator`

The application must also have the following Application API permissions for Microsoft Graph:

- `Application.ReadWrite.All`
- `Policy.Read.All`
- `User.Read.All`

Finally, in order to ensure that Azurerator may pre-approve delegated API permissions for the managed applications,
you will need to find and configure the `azure.permissiongrant-resource-id` configuration flag.

This ID is the _Object ID_ of an Azure AD Enterprise Application that is unique to each tenant. 

You will find this under either the name of `GraphAggregatorService` or `Microsoft Graph`.
Look for an Enterprise Application that has an _Application ID_ equal to `00000003-0000-0000-c000-000000000000`.

### Configuration

Set up the required environment variables as per the [config](./pkg/config/config.go).

```yaml
# ./azurerator.yaml

azure:
  auth:
    client-id: ""
    client-secret: ""
  tenant:
    id: "726d6769-7efc-4578-990a-f483ec2ec2d3"
    name: "local.test"
  permissiongrant-resource-id: ""
  features:
    claims-mapping-policies:
      enabled: false
      navident: ""
    teams-management:
      enabled: false
      service-principal-id: ""
    groups-assignment:
      enabled: false
      all-users-group-id: ""
validations:
  tenant:
    required: false
cluster-name: local
debug: true
secret-rotation:
  max-age: 168h
```

Then, assuming you have a Kubernetes cluster running locally (e.g.
using [minikube](https://github.com/kubernetes/minikube)):

```shell script
ulimit -n 4096  # for controller-gen
make run
kubectl apply -f ./config/samples/AzureAdApplication.yaml
```
