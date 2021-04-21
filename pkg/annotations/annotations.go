package annotations

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	DeleteKey             = "azure.nais.io/delete"
	NotInTeamNamespaceKey = "azure.nais.io/not-in-team-namespace"
	ResynchronizeKey      = "azure.nais.io/resync"
	RotateKey             = "azure.nais.io/rotate"
)

func SetAnnotation(resource v1.ObjectMetaAccessor, key, value string) {
	a := resource.GetObjectMeta().GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[key] = value
	resource.GetObjectMeta().SetAnnotations(a)
}

func HasAnnotation(resource v1.ObjectMetaAccessor, key string) (string, bool) {
	value, found := resource.GetObjectMeta().GetAnnotations()[key]
	return value, found
}

func RemoveAnnotation(resource v1.ObjectMetaAccessor, key string) {
	_, found := HasAnnotation(resource, key)
	if found {
		a := resource.GetObjectMeta().GetAnnotations()
		delete(a, key)
		resource.GetObjectMeta().SetAnnotations(a)
	}
}
