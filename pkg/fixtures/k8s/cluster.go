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
	unusedSecretKey := client.ObjectKey{
		Namespace: c.Namespace,
		Name:      c.UnusedSecretName,
	}

	resources := map[client.ObjectKey]runtime.Object{
		key:             &v1alpha1.AzureAdApplication{},
		key:             &corev1.Pod{},
		unusedSecretKey: &corev1.Secret{},
	}
	timeout := time.NewTimer(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-timeout.C:
			return fmt.Errorf("timeout while waiting for cluster fixtures setup synchronization")
		case <-ticker.C:
			exists, err := allExists(ctx, cli, resources)
			if err != nil {
				return err
			}
			if exists {
				return nil
			}
		}
	}
}

func allExists(ctx context.Context, cli client.Client, resources map[client.ObjectKey]runtime.Object) (bool, error) {
	for key, resource := range resources {
		err := cli.Get(ctx, key, resource)
		if err == nil {
			continue
		}
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
