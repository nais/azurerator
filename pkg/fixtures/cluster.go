package fixtures

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/azureator/pkg/labels"
)

type ClusterFixtures struct {
	client.Client
	Config
	azureAdApplication *v1.AzureAdApplication
	pod                *corev1.Pod
	podEnvFrom         *corev1.Pod
	unusedSecret       *corev1.Secret
}

type Config struct {
	AzureAppName     string
	NamespaceName    string
	SecretName       string
	UnusedSecretName string
}

type resource struct {
	client.ObjectKey
	client.Object
}

func New(cli client.Client, config Config) ClusterFixtures {
	return ClusterFixtures{Client: cli, Config: config}
}

func (c ClusterFixtures) WithAzureApp() ClusterFixtures {
	key := types.NamespacedName{
		Namespace: c.NamespaceName,
		Name:      c.AzureAppName,
	}
	spec := v1.AzureAdApplicationSpec{
		ReplyUrls: []v1.AzureAdReplyUrl{
			{
				Url: "http://localhost:3000/auth/callback",
			},
		},
		PreAuthorizedApplications: []v1.AccessPolicyInboundRule{
			{
				AccessPolicyRule: v1.AccessPolicyRule{
					Application: "some-other-app",
					Namespace:   key.Namespace,
					Cluster:     "test-cluster",
				},
			},
		},
		LogoutUrl:  "",
		SecretName: c.SecretName,
	}
	c.azureAdApplication = &v1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:        key.Name,
			Namespace:   key.Namespace,
			ClusterName: "test-cluster",
		},
		Spec: spec,
	}
	return c
}

func (c ClusterFixtures) WithUnusedSecret() ClusterFixtures {
	c.unusedSecret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.UnusedSecretName,
			Namespace: c.NamespaceName,
			Labels: map[string]string{
				labels.AppLabelKey:  c.AzureAppName,
				labels.TypeLabelKey: labels.TypeLabelValue,
			},
		},
	}
	return c
}

func (c ClusterFixtures) WithPods() ClusterFixtures {
	c.pod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.AzureAppName,
			Namespace: c.NamespaceName,
			Labels: map[string]string{
				labels.AppLabelKey: c.AzureAppName,
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
	c.podEnvFrom = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-envfrom", c.AzureAppName),
			Namespace: c.NamespaceName,
			Labels: map[string]string{
				labels.AppLabelKey: c.AzureAppName,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: "foo",
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: c.SecretName,
								},
							},
						},
					},
				},
			},
		},
	}
	return c
}

func (c ClusterFixtures) WithTenant(tenant string) ClusterFixtures {
	c.azureAdApplication.Spec.Tenant = tenant
	return c
}

func (c ClusterFixtures) WithMinimalConfig() ClusterFixtures {
	return c.WithAzureApp().
		WithPods().
		WithUnusedSecret()
}

func (c ClusterFixtures) Setup() error {
	resources := make([]resource, 0)

	ctx := context.Background()

	err := c.Create(ctx, namespace(c.NamespaceName))
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	if c.pod != nil {
		err := c.Create(ctx, c.pod)
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}

		resources = append(resources, resource{
			ObjectKey: client.ObjectKey{
				Namespace: c.NamespaceName,
				Name:      c.AzureAppName,
			},
			Object: &corev1.Pod{},
		})
	}

	if c.podEnvFrom != nil {
		err := c.Create(ctx, c.podEnvFrom)
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}

		resources = append(resources, resource{
			ObjectKey: client.ObjectKey{
				Namespace: c.NamespaceName,
				Name:      fmt.Sprintf("%s-envfrom", c.AzureAppName),
			},
			Object: &corev1.Pod{},
		})
	}

	if c.unusedSecret != nil {
		err := c.Create(ctx, c.unusedSecret)
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	if c.azureAdApplication != nil {
		err := c.Create(ctx, c.azureAdApplication)
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}

		resources = append(resources, resource{
			ObjectKey: client.ObjectKey{
				Namespace: c.NamespaceName,
				Name:      c.AzureAppName,
			},
			Object: &v1.AzureAdApplication{},
		})
	}

	return c.waitForClusterResources(ctx, resources)
}

func (c ClusterFixtures) waitForClusterResources(ctx context.Context, resources []resource) error {
	timeout := time.NewTimer(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-timeout.C:
			return fmt.Errorf("timeout while waiting for cluster fixtures setup synchronization")
		case <-ticker.C:
			exists, err := allExists(ctx, c.Client, resources)
			if err != nil {
				return err
			}
			if exists {
				return nil
			}
		}
	}
}

func allExists(ctx context.Context, cli client.Client, resources []resource) (bool, error) {
	for _, resource := range resources {
		err := cli.Get(ctx, resource.ObjectKey, resource.Object)
		if err == nil {
			continue
		}
		if errors.IsNotFound(err) {
			log.Infof("%+v not found", resource.ObjectKey)
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func namespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
