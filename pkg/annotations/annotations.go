package annotations

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PreserveKey           = "azure.nais.io/preserve"
	NotInTeamNamespaceKey = "azure.nais.io/not-in-team-namespace"
	ResynchronizeKey      = "azure.nais.io/resync"
	RotateKey             = "azure.nais.io/rotate"
	StakaterReloaderKey   = "reloader.stakater.com/match"
)

func SetAnnotation(resource client.Object, key, value string) {
	a := resource.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[key] = value
	resource.SetAnnotations(a)
}

func HasAnnotation(resource client.Object, key string) (string, bool) {
	value, found := resource.GetAnnotations()[key]
	return value, found
}

func RemoveAnnotation(resource client.Object, key string) {
	_, found := HasAnnotation(resource, key)
	if found {
		a := resource.GetAnnotations()
		delete(a, key)
		resource.SetAnnotations(a)
	}
}
