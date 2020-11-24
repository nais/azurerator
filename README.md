# azurerator

Azurerator is a Kubernetes cluster operator for automated registration and lifecycle management of Azure Active Directory applications.

This specific implementation is tailored towards managing Azure AD applications within a single tenant for use in Web APIs,
i.e. both application and user authentication and authorization.

## For [NAIS](https://nais.io) end-users

See <https://doc.nais.io/security/auth/azure-ad>

## CRD

The operator introduces a new Kind `AzureAdApplication` (shortname `azureapp`), and acts upon changes to resources of this kind.

See the spec in [config/crd/bases/nais.io_azureadapplications.yaml](config/crd/nais.io_azureadapplications.yaml) for details.

An example resource is available in [config/samples/azureadapplication.yaml](./config/samples/azureadapplication.yaml).

## Lifecycle

![overview][overview]

See [lifecycle](./docs/lifecycle.md) for details.

[overview]: ./docs/sequence.png "Sequence diagram"

## Development

### Installation

```shell script
make install
```

### Configuration

Set up the required environment variables as per the [config](./pkg/config/config.go) 
and [Azure config](./pkg/azure/config/config.go):

```yaml
# ./azurerator.yaml

azure:
  auth:
    client-id: ""
    client-secret: ""
  tenant: ""
  permissiongrantresourceid: ""
  teamsmanagement:
    service-principal-id: "" 
cluster-name: local
debug: true
```

Then, assuming you have a Kubernetes cluster running locally (e.g. using [minikube](https://github.com/kubernetes/minikube)):

```shell script
ulimit -n 4096  # for controller-gen
make run
kubectl apply -f ./config/samples/AzureAdApplication.yaml
```
