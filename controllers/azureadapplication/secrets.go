package azureadapplication

import (
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/labels"
	"github.com/nais/azureator/pkg/secrets"
	"github.com/nais/liberator/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

type secretsClient struct {
	*transaction
	Reconciler
	secretName string
}

func (r Reconciler) secrets(transaction *transaction) secretsClient {
	return secretsClient{
		transaction: transaction,
		Reconciler:  r,
		secretName:  transaction.instance.Spec.SecretName,
	}
}

func (s secretsClient) CreateOrUpdate(result azure.ApplicationResult, set azure.CredentialsSet, azureOpenIDConfig config.AzureOpenIdConfig) error {
	s.log.Infof("processing secret with name '%s'...", s.secretName)

	objectMeta := kubernetes.ObjectMeta(s.secretName, s.instance.GetNamespace(), labels.Labels(s.instance))

	secret := &corev1.Secret{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		Type: corev1.SecretTypeOpaque,
	}

	stringData, err := secrets.SecretData(result, set, azureOpenIDConfig, s.secretDataKeys)
	if err != nil {
		return fmt.Errorf("while creating secret data: %w", err)
	}

	secretMutateFn := func() error {
		secret.StringData = stringData
		return ctrl.SetControllerReference(s.instance, secret, s.Scheme)
	}

	res, err := controllerutil.CreateOrUpdate(s.ctx, s.Client, secret, secretMutateFn)
	if err != nil {
		return fmt.Errorf("creating or updating secret %s: %w", s.secretName, err)
	}

	s.log.Infof("secret '%s' %s", s.secretName, res)
	return nil
}

func (s secretsClient) GetManaged() (*kubernetes.SecretLists, error) {
	// fetch all application pods for this app
	podList, err := kubernetes.ListPodsForApplication(s.ctx, s.Reader, s.instance.GetName(), s.instance.GetNamespace())
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	var allSecrets corev1.SecretList
	opts := []client.ListOption{
		client.InNamespace(s.instance.GetNamespace()),
		client.MatchingLabels(labels.Labels(s.instance)),
	}
	if err := s.Reader.List(s.ctx, &allSecrets, opts...); err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := kubernetes.ListUsedAndUnusedSecretsForPods(allSecrets, podList)
	return &podSecrets, nil
}

func (s secretsClient) DeleteUnused(unused corev1.SecretList) error {
	for i, oldSecret := range unused.Items {
		if oldSecret.Name == s.secretName {
			continue
		}
		s.log.Infof("deleting unused secret '%s'...", oldSecret.Name)
		if err := s.Client.Delete(s.ctx, &unused.Items[i]); err != nil {
			return fmt.Errorf("deleting unused secret: %w", err)
		}
	}
	return nil
}
