package annotations

import (
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PreserveKey         = "azure.nais.io/preserve"
	ResynchronizeKey    = "azure.nais.io/resync"
	RotateKey           = "azure.nais.io/rotate"
	StakaterReloaderKey = "reloader.stakater.com/match"
)

func SetAnnotation(resource client.Object, key, value string) {
	a := resource.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[key] = value
	resource.SetAnnotations(a)
}

// AddToAnnotation appends the value to the existing list of values for the given key, separated by commas.
// If there are no existing values, value itself is used.
func AddToAnnotation(resource client.Object, key, value string) {
	a := resource.GetAnnotations()
	if a == nil {
		SetAnnotation(resource, key, value)
		return
	}

	existingValue, ok := a[key]
	if ok {
		a[key] = strings.Join([]string{existingValue, value}, ",")
	} else {
		a[key] = value
	}

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

// RemoveFromAnnotation removes the first element from the annotation value, if there are multiple comma-separated values within the value.
func RemoveFromAnnotation(resource client.Object, key string) {
	existingValue, found := HasAnnotation(resource, key)
	if !found {
		return
	}

	a := resource.GetAnnotations()

	existingValues := strings.Split(existingValue, ",")
	if len(existingValues) > 1 {
		a[key] = strings.Join(existingValues[1:], ",")
	} else {
		delete(a, key)
	}

	resource.SetAnnotations(a)
}
