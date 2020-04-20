package resourcecreator

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (c Creator) createSecretSpec() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: c.CreateObjectMeta(),
	}
}

func (c Creator) createSecretMutateFn(secret *corev1.Secret) controllerutil.MutateFn {
	return func() error {
		secret.StringData = c.toSecretData()
		secret.Type = corev1.SecretTypeOpaque
		return nil
	}
}

func (c Creator) toSecretData() map[string]string {
	return map[string]string{
		"clientId":     c.Application.Credentials.Private.ClientId,
		"clientSecret": c.Application.Credentials.Private.ClientSecret,
	}
}
