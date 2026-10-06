package secrets

import (
	"testing"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/fake"
	"github.com/nais/azureator/pkg/azure/result"
)

// The extracted certificates must equal the bytes that Add and Rotate sent to Graph.
func TestGetCertificates(t *testing.T) {
	app := &v1.AzureAdApplication{}
	app.SetName("test-app")
	app.SetNamespace("test")
	keys := NewSecretDataKeys()
	used := fake.AzureCredentialsSet(app, "test-cluster")
	unused := fake.AzureCredentialsSet(app, "test-cluster")

	secret := func(set credentials.Set) corev1.Secret {
		data, err := SecretData(fake.AzureApplicationResult(app, result.OperationCreated), set, fake.AzureOpenIdConfig(), keys)
		require.NoError(t, err)
		byteData := make(map[string][]byte, len(data))
		for k, v := range data {
			byteData[k] = []byte(v)
		}
		return corev1.Secret{Data: byteData}
	}

	certificates := NewExtractor(kubernetes.SecretLists{
		Used:   corev1.SecretList{Items: []corev1.Secret{secret(used)}},
		Unused: corev1.SecretList{Items: []corev1.Secret{secret(unused)}},
	}, keys).GetCertificates()

	assert.Equal(t, map[string][]byte{
		used.Current.Certificate.KeyId:   used.Current.Certificate.Jwk.PublicPem,
		used.Next.Certificate.KeyId:      used.Next.Certificate.Jwk.PublicPem,
		unused.Current.Certificate.KeyId: unused.Current.Certificate.Jwk.PublicPem,
		unused.Next.Certificate.KeyId:    unused.Next.Certificate.Jwk.PublicPem,
	}, certificates)
}
