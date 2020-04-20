package resourcecreator

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (c Creator) createConfigMapSpec() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: c.CreateObjectMeta(),
	}
}

func (c Creator) createConfigMapMutateFn(configMap *corev1.ConfigMap) controllerutil.MutateFn {
	return func() error {
		configMap.Data = c.toConfigMapData()
		return nil
	}
}

func (c Creator) toConfigMapData() map[string]string {
	return map[string]string{
		"clientId": c.Application.Credentials.Public.ClientId,
	}
}
