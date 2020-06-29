package pods

import (
	"context"

	v1 "github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetForApplication(ctx context.Context, instance *v1.AzureAdApplication, reader client.Reader) (*corev1.PodList, error) {
	selector := client.MatchingLabels{
		labels.AppLabelKey: instance.GetName(),
	}
	namespace := client.InNamespace(instance.GetNamespace())
	podList := &corev1.PodList{}
	err := reader.List(ctx, podList, selector, namespace)
	if err != nil {
		return nil, err
	}
	return podList, nil
}
