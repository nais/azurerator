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
	Reconciler
}

func (r Reconciler) secrets() secretsClient {
	return secretsClient{Reconciler: r}
}

func (s secretsClient) createOrUpdate(
	tx transaction,
	result azure.ApplicationResult,
	set azure.CredentialsSet,
	azureOpenIDConfig config.AzureOpenIdConfig,
	secretName string,
) error {
	objectMeta := kubernetes.ObjectMeta(secretName, tx.instance.GetNamespace(), labels.Labels(tx.instance))

	secret := &corev1.Secret{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		Type: corev1.SecretTypeOpaque,
	}

	stringData, err := secrets.SecretData(result, set, azureOpenIDConfig, tx.secretDataKeys)
	if err != nil {
		return fmt.Errorf("while creating secret data: %w", err)
	}

	secretMutateFn := func() error {
		secret.StringData = stringData
		return ctrl.SetControllerReference(tx.instance, secret, s.Scheme)
	}

	res, err := controllerutil.CreateOrUpdate(tx.ctx, s.Client, secret, secretMutateFn)
	if err != nil {
		return fmt.Errorf("creating or updating secret %s: %w", secretName, err)
	}

	tx.log.Infof("secret '%s' %s", secretName, res)
	return nil
}

func (s secretsClient) getManaged(tx transaction) (*kubernetes.SecretLists, error) {
	// fetch all application pods for this app
	podList, err := kubernetes.ListPodsForApplication(tx.ctx, s.Reader, tx.instance.GetName(), tx.instance.GetNamespace())
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	var allSecrets corev1.SecretList
	opts := []client.ListOption{
		client.InNamespace(tx.instance.GetNamespace()),
		client.MatchingLabels(labels.Labels(tx.instance)),
	}
	if err := s.Reader.List(tx.ctx, &allSecrets, opts...); err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := kubernetes.ListUsedAndUnusedSecretsForPods(allSecrets, podList)
	return &podSecrets, nil
}

func (s secretsClient) deleteUnused(tx transaction, unused corev1.SecretList) error {
	for i, oldSecret := range unused.Items {
		if oldSecret.Name == tx.instance.Spec.SecretName {
			continue
		}
		tx.log.Infof("deleting unused secret '%s'...", oldSecret.Name)
		if err := s.Client.Delete(tx.ctx, &unused.Items[i]); err != nil {
			return fmt.Errorf("deleting unused secret: %w", err)
		}
	}
	return nil
}

func (s secretsClient) process(tx transaction, applicationResult *azure.ApplicationResult) error {
	secretName := tx.instance.Spec.SecretName

	managedSecrets, err := s.getManaged(tx)
	if err != nil {
		return fmt.Errorf("getting managed secrets: %w", err)
	}

	secretsExtractor := secrets.NewExtractor(*managedSecrets, tx.secretDataKeys)

	keyIdsInUse := func() azure.KeyIdsInUse {
		keyIdsInUse := secretsExtractor.GetKeyIdsInUse()
		tx.instance.Status.CertificateKeyIds = keyIdsInUse.Certificate
		tx.instance.Status.PasswordKeyIds = keyIdsInUse.Password
		return keyIdsInUse
	}()

	credentialsSet, validCredentials, err := secretsExtractor.GetPreviousCredentialsSet(tx.instance.Status.SynchronizationSecretName)
	if err != nil {
		return fmt.Errorf("extracting credentials set from secret: %w", err)
	}

	// invalidate credentials if cluster resource status/spec conditions are not met
	validCredentials = validCredentials && tx.options.Process.Secret.Valid

	// ensure that existing credentials set are in sync with Azure
	if validCredentials {
		validInAzure, err := s.azure().validateCredentials(tx, *credentialsSet)
		if err != nil {
			return fmt.Errorf("validating azure credentials: %w", err)
		}

		validCredentials = validCredentials && validInAzure
	}

	// return early if no operations needed
	if validCredentials && !tx.options.Process.Secret.Rotate && applicationResult.IsNotModified() {
		return nil
	}

	switch {
	case !validCredentials:
		credentialsSet, keyIdsInUse, err = s.azure().addCredentials(tx, keyIdsInUse)
		if err != nil {
			return fmt.Errorf("adding azure credentials: %w", err)
		}
	case tx.options.Process.Secret.Rotate:
		credentialsSet, keyIdsInUse, err = s.azure().rotateCredentials(tx, *credentialsSet, keyIdsInUse)
		if err != nil {
			return fmt.Errorf("rotating azure credentials: %w", err)
		}
	}

	tx.log.Infof("processing secret with name '%s'...", secretName)
	if err := s.createOrUpdate(tx, *applicationResult, *credentialsSet, s.AzureOpenIDConfig, secretName); err != nil {
		return err
	}

	if err := s.deleteUnused(tx, managedSecrets.Unused); err != nil {
		return err
	}

	if !validCredentials || tx.options.Process.Secret.Rotate {
		tx.instance.Status.CertificateKeyIds = keyIdsInUse.Certificate
		tx.instance.Status.PasswordKeyIds = keyIdsInUse.Password
		now := metav1.Now()
		tx.instance.Status.SynchronizationSecretRotationTime = &now
	}

	return nil
}
