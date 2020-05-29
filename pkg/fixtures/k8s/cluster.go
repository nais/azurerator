package k8s

import (
	"context"

	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/resourcecreator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterFixtures struct {
	Name             string
	SecretName       string
	UnusedSecretName string
	Namespace        string
}

func (p ClusterFixtures) Setup(cli client.Client) error {
	ctx := context.Background()
	pod := p.podFixture()
	if err := cli.Create(ctx, pod); err != nil {
		return err
	}

	secret := p.unusedSecretFixture()
	if err := cli.Create(ctx, secret); err != nil {
		return err
	}
	azureAdApplication := p.azureAdApplicationFixture()
	if err := cli.Create(ctx, azureAdApplication); err != nil {
		return err
	}
	return nil
}

func (p ClusterFixtures) azureAdApplicationFixture() *v1alpha1.AzureAdApplication {
	key := types.NamespacedName{
		Namespace: p.Namespace,
		Name:      p.Name,
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
		SecretName: p.SecretName,
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

func (p ClusterFixtures) unusedSecretFixture() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.UnusedSecretName,
			Namespace: p.Namespace,
			Labels: map[string]string{
				resourcecreator.AppLabelKey:  p.Name,
				resourcecreator.TypeLabelKey: resourcecreator.TypeLabelValue,
			},
		},
	}
}

func (p ClusterFixtures) podFixture() *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
			Labels: map[string]string{
				resourcecreator.AppLabelKey: p.Name,
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
							SecretName: p.SecretName,
						},
					},
				},
			},
		},
	}
}
