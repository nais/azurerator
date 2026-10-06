package secrets

import (
	"github.com/nais/liberator/pkg/kubernetes"

	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/secrets"
)

type Secrets struct {
	// Certificates holds public certificate PEM by Azure key ID.
	Certificates      map[string][]byte
	DataKeys          secrets.SecretDataKeys
	KeyIDs            credentials.KeyIDs
	LatestCredentials Credentials
	ManagedSecrets    kubernetes.SecretLists
}

type Credentials struct {
	Set   *credentials.Set
	Valid bool
}
