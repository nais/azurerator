package customresources

import (
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetUniqueName(in v1.AccessPolicyRule) string {
	return kubernetes.UniformResourceName(&metav1.ObjectMeta{
		Name:        in.Application,
		Namespace:   in.Namespace,
		ClusterName: in.Cluster,
	})
}
