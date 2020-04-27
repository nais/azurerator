package resourcecreator

import (
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	LabelType string = "azurerator.nais.io"
)

type Creator interface {
	Spec() (runtime.Object, error)
	MutateFn(object runtime.Object) (controllerutil.MutateFn, error)
	Name() string
}

type DefaultCreator struct {
	Credential  v1alpha1.AzureAdCredential
	Application azure.Application
}

func (c DefaultCreator) ObjectMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: c.Credential.Namespace,
		Labels:    c.Labels(),
	}
}

func (c DefaultCreator) Labels() map[string]string {
	return map[string]string{
		"app":  c.Credential.Name,
		"type": LabelType,
	}
}