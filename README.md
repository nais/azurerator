# azurerator

Kubernetes cluster operator for automated registration of Azure Active Directory applications and accompanying service principals.

Creates or updates a resource in Azure AD when the `AzureAdCredential` (shortname `azuread`) resource is created, 
and in turn creates cluster resources (i.e. a `Secret` and a `ConfigMap`) with the necessary information for
applications to make use of Azure AD. Also handles garbage cleaning, i.e. the Azure AD application is deleted should
the equivalent `AzureAdCredential` resource be deleted by implementing a finalizer/pre-delete hook.

## Installation
```shell script
make install
```

## Development

Set up the required environment variables as per the [Azure config](./pkg/azure/config.go).

Then, assuming that you have a Kubernetes cluster running locally (e.g. using [minikube](https://github.com/kubernetes/minikube)):

```shell script
make run
kubectl apply -f ./config/samples/azureadcredential.yaml
```
