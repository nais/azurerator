package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/resourcecreator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterFixtures struct {
	Name             string
	SecretName       string
	UnusedSecretName string
	Namespace        string
}

func (c ClusterFixtures) Setup(cli client.Client) error {
	ctx := context.Background()
	pod := c.podFixture()
	if err := cli.Create(ctx, pod); err != nil {
		return err
	}

	secret := c.unusedSecretFixture()
	if err := cli.Create(ctx, secret); err != nil {
		return err
	}
	azureAdApplication := c.azureAdApplicationFixture()
	if err := cli.Create(ctx, azureAdApplication); err != nil {
		return err
	}
	return c.waitForClusterResources(ctx, cli)
}

func (c ClusterFixtures) azureAdApplicationFixture() *v1alpha1.AzureAdApplication {
	key := types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
	spec := v1alpha1.AzureAdApplicationSpec{
		ReplyUrls: []v1alpha1.AzureAdReplyUrl{
			{
				Url: "http://localhost:3000/auth/callback",
			},
		},
		PreAuthorizedApplications: []v1alpha1.AzureAdPreAuthorizedApplication{
			{
				Application: "some-other-app",
				Namespace:   key.Namespace,
				Cluster:     "test-cluster",
			},
		},
		LogoutUrl:  "",
		SecretName: c.SecretName,
	}
	return &v1alpha1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:        key.Name,
			Namespace:   key.Namespace,
			ClusterName: "test-cluster",
		},
		Spec: spec,
	}
}

func (c ClusterFixtures) unusedSecretFixture() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.UnusedSecretName,
			Namespace: c.Namespace,
			Labels: map[string]string{
				resourcecreator.AppLabelKey:  c.Name,
				resourcecreator.TypeLabelKey: resourcecreator.TypeLabelValue,
			},
		},
	}
}

func (c ClusterFixtures) podFixture() *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
			Labels: map[string]string{
				resourcecreator.AppLabelKey: c.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: "foo",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: c.SecretName,
						},
					},
				},
			},
		},
	}
}

func (c ClusterFixtures) waitForClusterResources(ctx context.Context, cli client.Client) error {
	key := client.ObjectKey{
		Namespace: c.Namespace,
		Name:      c.Name,
	}

	resources := []runtime.Object{
		&v1alpha1.AzureAdApplication{},
		&corev1.Pod{},
		&corev1.Secret{},
	}
	timeout := time.NewTimer(30 * time.Second)
	ticker := time.NewTicker(250 * time.Millisecond)

	for {
		select {
		case <-timeout.C:
			return fmt.Errorf("timeout while waiting for cluster fixtures setup synchronization")
		case <-ticker.C:
			return getAllOrError(ctx, cli, key, resources)
		}
	}
}

func getAllOrError(ctx context.Context, cli client.Client, key client.ObjectKey, resources []runtime.Object) error {
	for _, resource := range resources {
		err := cli.Get(ctx, key, resource)
		if err == nil {
			return nil
		}
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
