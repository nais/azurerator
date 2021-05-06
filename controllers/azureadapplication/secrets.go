package azureadapplication

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/labels"
	"github.com/nais/azureator/pkg/secrets"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
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

type transactionSecrets struct {
	credentials    credentials
	dataKeys       secrets.SecretDataKeys
	keyIdsInUse    azure.KeyIdsInUse
	managedSecrets kubernetes.SecretLists
}

type credentials struct {
	set   *azure.CredentialsSet
	valid bool
}

func (r Reconciler) secrets() secretsClient {
	return secretsClient{Reconciler: r}
}

func (s secretsClient) prepare(ctx context.Context, instance *v1.AzureAdApplication) (*transactionSecrets, error) {
	dataKeys := secrets.NewSecretDataKeys(instance.Spec.SecretKeyPrefix)

	managedSecrets, err := s.getManaged(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("getting managed secrets: %w", err)
	}

	secretsExtractor := secrets.NewExtractor(*managedSecrets, dataKeys)

	keyIdsInUse := func() azure.KeyIdsInUse {
		keyIdsInUse := secretsExtractor.GetKeyIdsInUse()
		instance.Status.CertificateKeyIds = keyIdsInUse.Certificate
		instance.Status.PasswordKeyIds = keyIdsInUse.Password
		return keyIdsInUse
	}()

	credentialsSet, validCredentials, err := secretsExtractor.GetPreviousCredentialsSet(instance.Status.SynchronizationSecretName)
	if err != nil {
		return nil, fmt.Errorf("extracting credentials set from secret: %w", err)
	}

	return &transactionSecrets{
		credentials: credentials{
			set:   credentialsSet,
			valid: validCredentials,
		},
		dataKeys:       dataKeys,
		keyIdsInUse:    keyIdsInUse,
		managedSecrets: *managedSecrets,
	}, nil
}

func (s secretsClient) process(tx transaction, applicationResult *azure.ApplicationResult) error {
	// return early if no operations needed
	if tx.options.Process.Secret.Valid && !tx.options.Process.Secret.Rotate && applicationResult.IsNotModified() {
		return nil
	}

	var err error

	credentialsSet := tx.secrets.credentials.set
	keyIdsInUse := tx.secrets.keyIdsInUse

	switch {
	case !tx.options.Process.Secret.Valid:
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

	if err := s.createOrUpdate(tx, *applicationResult, *credentialsSet); err != nil {
		return err
	}

	if err := s.deleteUnused(tx); err != nil {
		return err
	}

	if !tx.options.Process.Secret.Valid || tx.options.Process.Secret.Rotate {
		tx.instance.Status.CertificateKeyIds = keyIdsInUse.Certificate
		tx.instance.Status.PasswordKeyIds = keyIdsInUse.Password
		now := metav1.Now()
		tx.instance.Status.SynchronizationSecretRotationTime = &now
	}

	return nil
}

func (s secretsClient) createOrUpdate(tx transaction, result azure.ApplicationResult, set azure.CredentialsSet) error {
	secretName := tx.instance.Spec.SecretName
	objectMeta := kubernetes.ObjectMeta(secretName, tx.instance.GetNamespace(), labels.Labels(tx.instance))

	secret := &corev1.Secret{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		Type: corev1.SecretTypeOpaque,
	}

	stringData, err := secrets.SecretData(result, set, s.AzureOpenIDConfig, tx.secrets.dataKeys)
	if err != nil {
		return fmt.Errorf("while creating secret data for secret '%s': %w", secretName, err)
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

func (s secretsClient) getManaged(ctx context.Context, instance *v1.AzureAdApplication) (*kubernetes.SecretLists, error) {
	// fetch all application pods for this app
	podList, err := kubernetes.ListPodsForApplication(ctx, s.Reader, instance.GetName(), instance.GetNamespace())
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	var allSecrets corev1.SecretList
	opts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(labels.Labels(instance)),
	}
	if err := s.Reader.List(ctx, &allSecrets, opts...); err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := kubernetes.ListUsedAndUnusedSecretsForPods(allSecrets, podList)
	return &podSecrets, nil
}

func (s secretsClient) deleteUnused(tx transaction) error {
	unused := tx.secrets.managedSecrets.Unused

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
