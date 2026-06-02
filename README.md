# Azurerator

Kubernetes operator for declarative lifecycle management of Entra ID (formerly Azure AD) applications.

## How it works

Azurerator watches for `AzureAdApplication` (`azureapp`) resources and reconciles them against Entra ID — registering apps, configuring credentials, managing pre-authorized clients, and producing Kubernetes Secrets with the resulting metadata.

See [Lifecycle](docs/lifecycle.md) for the full sequence diagram and detailed operations.

### Example resource

```yaml
apiVersion: nais.io/v1
kind: AzureAdApplication
metadata:
  name: myapp
  namespace: myteam
spec:
  secretName: azuread-myapp
  preAuthorizedApplications:
    - application: other-app
      namespace: other-team
      cluster: other-cluster
  replyUrls:
    - url: "https://myapp.example.com/oauth2/callback"
```

See the [Custom Resource Definition (CRD)](https://github.com/nais/liberator/blob/main/config/crd/bases/nais.io_azureadapplications.yaml) for all available options.

### Secret keys

The operator produces a Kubernetes Secret using the name specified in `.spec.secretName`.
The Secret contains the following keys:

| Key                                  | Description                                                                                            |
|--------------------------------------|--------------------------------------------------------------------------------------------------------|
| `AZURE_APP_CLIENT_ID`                | Application (client) ID                                                                                |
| `AZURE_APP_CLIENT_SECRET`            | Client secret (password credential)                                                                    |
| `AZURE_APP_JWK`                      | Private key ([JWK](https://datatracker.ietf.org/doc/html/rfc7517#section-4)) for client assertion      |
| `AZURE_APP_JWKS`                     | Private key set ([JWKS](https://datatracker.ietf.org/doc/html/rfc7517#section-5)) for client assertion |
| `AZURE_APP_WELL_KNOWN_URL`           | Endpoint to OpenID Connect discovery document                                                          |
| `AZURE_OPENID_CONFIG_ISSUER`         | `issuer` from discovery document                                                                       |
| `AZURE_OPENID_CONFIG_JWKS_URI`       | `jwks_uri` from discovery document                                                                     |
| `AZURE_OPENID_CONFIG_TOKEN_ENDPOINT` | `token_endpoint` from discovery document                                                               |

## Documentation

| Document                                                                                                              | Description                                                         |
|-----------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------|
| [Lifecycle](docs/lifecycle.md)                                                                                        | Detailed walkthrough of all operations performed per reconciliation |
| [Configuration](docs/configuration.md)                                                                                | Entra ID setup, all flags, and example config                       |
| [CRD spec (liberator)](https://github.com/nais/liberator/blob/main/config/crd/bases/nais.io_azureadapplications.yaml) | Full custom resource definition                                     |
| [Example resource](config/samples/azureadapplication.yaml)                                                            | Sample `AzureAdApplication` manifest                                |

## Development

### Requirements

- [mise](https://mise.jdx.dev/) — tool version manager and task runner

### Getting started

Start a local Kubernetes cluster, e.g. using [kind](https://kind.sigs.k8s.io/), [minikube](https://minikube.sigs.k8s.io/), or similar.

```shell
mise install              # install prerequisites
mise run install:crd      # install CRDs into cluster
mise run local            # start the controller

# in another shell
mise run install:sample   # apply a sample AzureAdApplication resource
```

### Testing

```shell
mise run test
```
