package secret

import (
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/resourcecreator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Creator struct {
	resourcecreator.DefaultCreator
}

func New(credential v1alpha1.AzureAdCredential, application azure.Application) resourcecreator.Creator {
	return Creator{
		resourcecreator.DefaultCreator{
			Credential:  credential,
			Application: application,
		},
	}
}

func (c Creator) Spec() (runtime.Object, error) {
	return &corev1.Secret{
		ObjectMeta: c.CreateObjectMeta(),
	}, nil
}

func (c Creator) MutateFn(object runtime.Object) (controllerutil.MutateFn, error) {
	secret := object.(*corev1.Secret)
	return func() error {
		secret.StringData = c.toSecretData()
		secret.Type = corev1.SecretTypeOpaque
		return nil
	}, nil
}

func (c Creator) toSecretData() map[string]string {
	return map[string]string{
		"clientId":     c.Application.Credentials.Private.ClientId,
		"clientSecret": c.Application.Credentials.Private.ClientSecret,
	}
}
