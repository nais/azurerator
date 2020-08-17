package namespaces

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=*,resources=namespaces,verbs=get;list;watch

func GetShared(ctx context.Context, reader client.Reader) (corev1.NamespaceList, error) {
	var namespaces corev1.NamespaceList
	mLabels := client.MatchingLabels{
		"shared": "true",
	}
	if err := reader.List(ctx, &namespaces, mLabels); err != nil {
		return namespaces, fmt.Errorf("failed to get list of shared namespaces: %w", err)
	}
	return namespaces, nil
}

func GetAll(ctx context.Context, reader client.Reader) (corev1.NamespaceList, error) {
	var namespaces corev1.NamespaceList
	if err := reader.List(ctx, &namespaces); err != nil {
		return namespaces, fmt.Errorf("failed to get list namespaces: %w", err)
	}
	return namespaces, nil
}