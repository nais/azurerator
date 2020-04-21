package resourcecreator

import (
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ResourcePrefix string = "azuread"
	LabelType      string = "azurerator.nais.io"
)

type Creator interface {
	Spec() (runtime.Object, error)
	MutateFn(object runtime.Object) (controllerutil.MutateFn, error)
}

type DefaultCreator struct {
	Credential  v1alpha1.AzureAdCredential
	Application azure.Application
}

func (c DefaultCreator) GetResourcePrefix() string {
	return ResourcePrefix
}

func (c DefaultCreator) CreateObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      c.CreateName(),
		Namespace: c.Credential.Namespace,
		Labels:    c.CreateLabels(),
	}
}

func (c DefaultCreator) CreateName() string {
	return fmt.Sprintf("%s-%s", c.GetResourcePrefix(), c.Credential.Name)
}

func (c DefaultCreator) CreateLabels() map[string]string {
	return map[string]string{
		"app":  c.Credential.Name,
		"type": LabelType,
	}
}
