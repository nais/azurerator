<!-- omit in toc -->
# azurerator

Azurerator is a Kubernetes cluster operator for automated registration and lifecycle management of Azure Active Directory applications.

This specific implementation is tailored towards managing Azure AD applications within a single tenant for use in Web APIs,
i.e. both application and user authentication and authorization.

<!-- omit in toc -->
## Table of Contents

- [1. Installation](#1-installation)
- [2. Development](#2-development)
- [3. CRD](#3-crd)
- [4. Lifecycle](#4-lifecycle)
  - [4.1 New applications](#41-new-applications)
    - [4.1.1 Display Name](#411-display-name)
    - [4.1.2 Authentication Platform](#412-authentication-platform)
      - [Application Identifier URI](#application-identifier-uri)
      - [OAuth2 Permission Scopes](#oauth2-permission-scopes)
      - [Redirect URIs (optional)](#redirect-uris-optional)
      - [Logout URLs (optional)](#logout-urls-optional)
      - [Application Roles](#application-roles)
    - [4.1.3 (Pre-)Authorized Client Applications](#413-pre-authorized-client-applications)
    - [4.1.4 Service Principal](#414-service-principal)
    - [4.1.5 Delegated Permissions](#415-delegated-permissions)
    - [4.1.6 Credentials](#416-credentials)
  - [4.2 Existing applications](#42-existing-applications)
    - [4.2.1 Credential Rotation](#421-credential-rotation)
  - [4.3 Cluster Resources](#43-cluster-resources)
    - [4.3.1 Secret](#431-secret)
  - [4.4 Deletion](#44-deletion)

## 1. Installation

```shell script
make install
```

## 2. Development

Set up the required environment variables as per the [config](./pkg/config/config.go) and [Azure config](./pkg/azure/config/config.go).

Then, assuming you have a Kubernetes cluster running locally (e.g. using [minikube](https://github.com/kubernetes/minikube)):

```shell script
ulimit -n 4096  # for controller-gen
make run
kubectl apply -f ./config/samples/AzureAdApplication.yaml
```

## 3. CRD

The operator introduces a new Kind `AzureAdApplication` (shortname `azureapp`), and acts upon changes to resources of this kind.

See the spec in [config/crd/bases/nais.io_azureadapplications.yaml](config/crd/nais.io_azureadapplications.yaml) for details.

An example resource is available in [config/samples/azureadapplication.yaml](./config/samples/azureadapplication.yaml).

## 4. Lifecycle

Whenever a `AzureAdApplication` resource is created or changed in the cluster, the operator will accordingly create or update
the equivalent resources in Azure AD to reflect the desired state.

The following is a short overview of operations performed.

### 4.1 New applications

Applications that do not exist in Azure AD will be registered with the following configuration:

#### 4.1.1 Display Name

The application in Azure AD will be assigned a display name of the following format:

```
<ClusterName>:<Namespace>:<Metadata.Name>
```

#### 4.1.2 Authentication Platform

By default, a **Web API** is registered as the authentication platform for the application, allowing for usage in _OIDC/OAuth2_ authentication flows with Azure AD.
This means the application can be used for both accessing and exposing Web APIs,
handling both end-user logins with OIDC and on-behalf-of flows and/or act as daemons for service-to-service communication.

##### Application Identifier URI

The Application Identifier URI uniquely identifies the Web app within the Azure AD tenant.

It is represented in the following form:
```
api://<clientId>
```

where `clientId` is the Azure Application / Client ID, e.g. `api://4f6fae71-89da-46ff-a6d5-04d27d76eb1a`.

Other applications may use this identifier within the when requesting access tokens for the application from Azure,
e.g. by providing the scope `api://<clientId>/.default` in the request.

##### OAuth2 Permission Scopes

A default set of [OAuth2 permission scopes](https://docs.microsoft.com/en-us/graph/api/resources/permissionscope?view=graph-rest-1.0) are registered for the application. 
These are exposed to client applications.

##### Redirect URIs (optional)

Redirect URIs are URIs that the Authorization Server will accept as destinations when returning authentication responses (tokens) after successfully authenticating users. Often referred to as reply URLs.

These are registered according to the list of URIs defined in the `AzureAdApplication` resource, i.e. `[]Spec.ReplyUrl`.

See <https://docs.microsoft.com/en-gb/azure/active-directory/develop/reply-url> for restrictions and limitations.

##### Logout URLs (optional)

`Spec.LogoutUrl` defines the `post_logout_redirect_uri` that Azure should redirect to after sign-out in order to properly implement single-sign-out.

See <https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-protocols-oidc#send-a-sign-out-request> for details.

##### Application Roles

An Application Role (AppRole) can be used to enforce authorization in the application.
The operator automatically registers an AppRole called `access_as_application`.

This enables an additional option for authorization checks for service-to-service calls,
where the receiving client API should validate that the calling application has been assigned the role and thus
been given access to this application.

The role should be present in the `roles` claim within the access token obtained from the OAuth2 client credentials flow.

#### 4.1.3 (Pre-)Authorized Client Applications

Pre-authorized client applications define the set of client applications allowed to perform 
[on-behalf-of flow](https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-on-behalf-of-flow)
to obtain access tokens intended for the application.

These are registered according to the list of applications defined in `[]Spec.PreAuthorizedApplication` in
the `AzureAdApplication` resource, with the following caveats:

- `Spec.PreAuthorizedApplication.Name` must follow the format `<ClusterName>:<Namespace>:<Metadata.Name>` in order to
correctly reference the intended application
- The operator will attempt to register these in a "best effort" manner.
- Any legitimate errors will be retried, however applications that do not exist will be skipped and not registered as a pre-authorized application.
- It is thus the user's responsibility to ensure that applications defined in the list of pre-authorized applications are (eventually) consistent.

The operator will register and grant admin consent for the **OAuth2 permission scopes** defined previously for all pre-authorized applications registered to the application.

Each pre-authorized application will also be assigned the **AppRole** described earlier.

#### 4.1.4 Service Principal

A [service principal](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals#service-principal-object) is registered and connected to the aforementioned application.

This enables us to register and automatically grant admin consent for delegated permissions for the application.

#### 4.1.5 Delegated Permissions

The operator will by default configure the application with the following permissions:

- `https://graph.microsoft.com/openid`
- `https://graph.microsoft.com/User.Read`
- `https://graph.microsoft.com/GroupMember.Read.All`

It will also automatically grant consent for these permissions to the application, 
allowing the application to perform sign-ins and reading basic user profile information without having to
prompt the end-user for manual consent.

See <https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-permissions-and-consent> for more details.

#### 4.1.6 Credentials

During application registration, a set of application secrets (or 'passwords') as well as self-signed certificates
are generated and assigned as valid authentication credentials for the application.

The validity for these are by default set to one (1) year,
however the operator also has built-in support for rotation of these credentials.

The unique identifiers (which can be looked up within Azure AD) for these keys are stored in the Status subresource for the resource,
i.e. in the fields:

- `Spec.Status.PasswordKeyId`
- `Spec.Status.CertificateKeyId`

These fields thus denote the currently used set of credentials.

See <https://docs.microsoft.com/en-us/azure/active-directory/develop/howto-create-service-principal-portal#certificates-and-secrets> for details.

### 4.2 Existing applications

If the application already exists in Azure AD, the operator will ensure that the configuration is up to date and that all 
the required Azure resources/configurations are in place analogously to the case of new applications described above.

Changes in configurable metadata such as:

- `[]Spec.PreAuthorizedApplication`
- `[]Spec.ReplyUrl`
- `Spec.LogoutUrl`

will result in updates to the application in Azure AD so that the desired state represented in the
resource is consistent with the actual state in Azure AD.

The associated cluster resources for the `AzureAdApplication` will also be updated accordingly.

#### 4.2.1 Credential Rotation

Whenever the `AzureAdApplication` resource changes, the operator will generate a new set of credentials and
associate these with the application.

In order to ensure zero downtime when rotating credentials, the following algorithm is used:

1. If the application only has a single set of registered credentials in Azure, then these will not be revoked.
2. The previous newest set of credentials will not be revoked.
3. Additionally, any set of credentials that exist in `corev1.Secret` resources in use by existing pods will not be revoked.
4. A new set of credentials are registered to the application in Azure AD
5. Any other key registered in Azure AD not matching the above will be revoked, i.e. any key deemed to be unused. 
6. The Status subresource is updated with the identifiers for the new set of credentials

### 4.3 Cluster Resources

The successful registration of the application in Azure AD will also produce cluster resources for the credentials
and other metadata that the application should use in order to authenticate itself to Azure AD.

#### 4.3.1 Secret

A `coreV1.Secret` with the name as defined in `Spec.SecretName` is created, containing the following keys and values:

#### `AZURE_APP_CLIENT_ID`

Azure AD client ID. Unique ID for the application in Azure AD.

Example value:

```e89006c5-7193-4ca3-8e26-d0990d9d981f```

#### `AZURE_APP_CLIENT_SECRET`

Azure AD client secret, i.e. password for [authenticating the application to Azure AD].

Example value:

```b5S0Bgg1OF17Ptpy4_uvUg-m.I~KU_.5RR```

#### `AZURE_APP_JWKS`

Private JWKS, i.e. containing a JWK with the private RSA key for creating signed JWTs when [authenticating to Azure AD with a certificate].

This will always contain a single key, i.e. the newest key registered.

Example value:

```json
{
  "keys": [
    {
      "use": "sig",
      "kty": "RSA",
      "kid": "jXDxKRE6a4jogcc4HgkDq3uVgQ0",
      "n": "xQ3chFsz...",
      "e": "AQAB",
      "d": "C0BVXQFQ...",
      "p": "9TGEF_Vk...",
      "q": "zb0yTkgqO...",
      "dp": "7YcKcCtJ...",
      "dq": "sXxLHp9A...",
      "qi": "QCW5VQjO...",
      "x5c": [
        "MIID8jCC..."
      ],
      "x5t": "jXDxKRE6a4jogcc4HgkDq3uVgQ0",
      "x5t#S256": "AH2gbUvjZYmSQXZ6-YIRxM2YYrLiZYW8NywowyGcxp0"
    }
  ]
}
```

#### `AZURE_APP_JWK`

Same as the above `AZURE_APP_JWKS`, just with the JWK unwrapped from the key set.

Example value:

```json
{
  "use": "sig",
  "kty": "RSA",
  "kid": "jXDxKRE6a4jogcc4HgkDq3uVgQ0",
  "n": "xQ3chFsz...",
  "e": "AQAB",
  "d": "C0BVXQFQ...",
  "p": "9TGEF_Vk...",
  "q": "zb0yTkgqO...",
  "dp": "7YcKcCtJ...",
  "dq": "sXxLHp9A...",
  "qi": "QCW5VQjO...",
  "x5c": [
    "MIID8jCC..."
  ],
  "x5t": "jXDxKRE6a4jogcc4HgkDq3uVgQ0",
  "x5t#S256": "AH2gbUvjZYmSQXZ6-YIRxM2YYrLiZYW8NywowyGcxp0"
}
```

#### `AZURE_APP_PRE_AUTHORIZED_APPS`

A JSON string. List of names and client IDs for the valid (i.e. those that exist in Azure AD) applications defined in `[]Spec.PreAuthorizedApplication`

Example value:

```
[
  {
    "name": "dev-gcp:othernamespace:app-a",
    "clientId": "381ce452-1d49-49df-9e7e-990ef0328d6c"
  },
  {
    "name": "dev-gcp:aura:app-b",
    "clientId": "048eb0e8-e18a-473a-a87d-dfede7c65d84"
  }
]
```

#### `AZURE_APP_WELL_KNOWN_URL`

The well-known URL with the tenant for which the application resides in.

Example value:

```
https://login.microsoftonline.com/77678b69-1daf-47b6-9072-771d270ac800/v2.0/.well-known/openid-configuration
```

### 4.4 Deletion

The operator implements a finalizer of type `finalizers.azurerator.nais.io` which will delete the application from Azure Active Directory
whenever the `AzureAdApplication` resource is deleted. 

OwnerReferences for the aforementioned cluster resources are also registered and should accordingly be garbage collected by the cluster.

[authenticating to Azure AD with a certificate]: https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-client-creds-grant-flow#second-case-access-token-request-with-a-certificate
[authenticating the application to Azure AD]: https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-client-creds-grant-flow#first-case-access-token-request-with-a-shared-secret
