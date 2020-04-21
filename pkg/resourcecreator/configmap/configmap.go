package configmap

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
	return &corev1.ConfigMap{
		ObjectMeta: c.CreateObjectMeta(),
	}, nil
}

func (c Creator) MutateFn(object runtime.Object) (controllerutil.MutateFn, error) {
	configMap := object.(*corev1.ConfigMap)
	return func() error {
		configMap.Data = c.toConfigMapData()
		return nil
	}, nil
}

func (c Creator) toConfigMapData() map[string]string {
	return map[string]string{
		"clientId": c.Application.Credentials.Public.ClientId,
	}
}
