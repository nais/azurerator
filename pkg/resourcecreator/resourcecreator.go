package resourcecreator

import (
	"fmt"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Creator struct {
	Credential  naisiov1alpha1.AzureAdCredential
	Application azure.Application
	Resource    runtime.Object
}

func (c Creator) GetResourcePrefix() string {
	return "azuread"
}

func (c Creator) CreateObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      c.CreateName(),
		Namespace: c.Credential.Namespace,
		Labels:    c.CreateLabels(),
	}
}

func (c Creator) CreateName() string {
	return fmt.Sprintf("%s-%s", c.GetResourcePrefix(), c.Credential.Name)
}

func (c Creator) CreateLabels() map[string]string {
	return map[string]string{
		"app":  c.Credential.Name,
		"type": "azurerator.nais.io",
	}
}

func (c Creator) CreateSpec() (runtime.Object, error) {
	var spec runtime.Object
	switch c.Resource.(type) {
	case *corev1.Secret:
		spec = c.createSecretSpec()
	case *corev1.ConfigMap:
		spec = c.createConfigMapSpec()
	default:
		return nil, fmt.Errorf("unsupported resource type %T", c.Resource)
	}
	return spec, nil
}

func (c Creator) CreateMutateFn(spec runtime.Object) (controllerutil.MutateFn, error) {
	switch orig := spec.(type) {
	case *corev1.Secret:
		return c.createSecretMutateFn(orig), nil
	case *corev1.ConfigMap:
		return c.createConfigMapMutateFn(orig), nil
	default:
		return nil, fmt.Errorf("unsupported resource type %T", c.Resource)
	}
}
