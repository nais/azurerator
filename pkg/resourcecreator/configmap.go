package resourcecreator

import (
	"encoding/json"
	"fmt"

	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ConfigMapCreator struct {
	DefaultCreator
}

func NewConfigMap(resource v1alpha1.AzureAdApplication, application azure.Application) Creator {
	return ConfigMapCreator{
		DefaultCreator{
			Resource:    resource,
			Application: application,
		},
	}
}

func (c ConfigMapCreator) Spec() (runtime.Object, error) {
	return &corev1.ConfigMap{
		ObjectMeta: c.ObjectMeta(c.Name()),
	}, nil
}

func (c ConfigMapCreator) MutateFn(object runtime.Object) (controllerutil.MutateFn, error) {
	configMap := object.(*corev1.ConfigMap)
	return func() error {
		data, err := c.toConfigMapData()
		if err != nil {
			return err
		}
		configMap.Data = data
		return nil
	}, nil
}

func (c ConfigMapCreator) Name() string {
	return c.Resource.Spec.ConfigMapName
}

func (c ConfigMapCreator) toConfigMapData() (map[string]string, error) {
	jwkJson, err := json.Marshal(c.Application.Credentials.Public.Jwk)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public JWK: %w", err)
	}
	// TODO - more user friendly format?
	preAuthAppsJson, err := json.Marshal(c.Application.PreAuthorizedApps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal preauthorized apps: %w", err)
	}
	return map[string]string{
		"clientId":          c.Application.Credentials.Public.ClientId,
		"jwks":              string(jwkJson),
		"preAuthorizedApps": string(preAuthAppsJson),
	}, nil
}
