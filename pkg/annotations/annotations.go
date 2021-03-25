package annotations

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	NotInTeamNamespaceKey = "azure.nais.io/not-in-team-namespace"
	DeleteKey             = "azure.nais.io/delete"
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
