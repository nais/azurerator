package resourcecreator

import (
	"fmt"

	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SecretCreator struct {
	DefaultCreator
}

func NewSecret(resource v1alpha1.AzureAdApplication, application azure.Application) Creator {
	return SecretCreator{
		DefaultCreator{
			Resource:    resource,
			Application: application,
		},
	}
}

func (c SecretCreator) Spec() (runtime.Object, error) {
	return &corev1.Secret{
		ObjectMeta: c.ObjectMeta(c.Name()),
	}, nil
}

func (c SecretCreator) MutateFn(object runtime.Object) (controllerutil.MutateFn, error) {
	secret := object.(*corev1.Secret)
	return func() error {
		data, err := c.toSecretData()
		if err != nil {
			return err
		}
		secret.StringData = data
		secret.Type = corev1.SecretTypeOpaque
		return nil
	}, nil
}

func (c SecretCreator) Name() string {
	return c.Resource.Spec.SecretName
}

func (c SecretCreator) toSecretData() (map[string]string, error) {
	jwkJson, err := c.Application.Credentials.Private.Jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private JWK: %w", err)
	}
	return map[string]string{
		"clientId":     c.Application.Credentials.Private.ClientId,
		"clientSecret": c.Application.Credentials.Private.ClientSecret,
		"jwk":          string(jwkJson),
	}, nil
}
