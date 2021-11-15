package secrets

import (
	"context"
	"fmt"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/labels"
	"github.com/nais/azureator/pkg/reconciler"
	"github.com/nais/azureator/pkg/secrets"
	"github.com/nais/azureator/pkg/transaction"
	transactionSecrets "github.com/nais/azureator/pkg/transaction/secrets"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

type secretsReconciler struct {
	reconciler.AzureAdApplication
	azureOpenIdConfig config.AzureOpenIdConfig
	client            client.Client
	reader            client.Reader
	scheme            *runtime.Scheme
}

func NewSecretsReconciler(
	azureAdApplication reconciler.AzureAdApplication,
	azureOpenIdConfig config.AzureOpenIdConfig,
	client client.Client,
	reader client.Reader,
	scheme *runtime.Scheme,
) reconciler.Secrets {
	return secretsReconciler{
		AzureAdApplication: azureAdApplication,
		azureOpenIdConfig:  azureOpenIdConfig,
		client:             client,
		reader:             reader,
		scheme:             scheme,
	}
}

func (s secretsReconciler) Prepare(ctx context.Context, instance *v1.AzureAdApplication) (*transactionSecrets.Secrets, error) {
	dataKeys := secrets.NewSecretDataKeys(instance.Spec.SecretKeyPrefix)

	managedSecrets, err := s.getManaged(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("getting managed secrets: %w", err)
	}

	secretsExtractor := secrets.NewExtractor(*managedSecrets, dataKeys)

	keyIdsInUse := func() credentials.KeyIdsInUse {
		keyIdsInUse := secretsExtractor.GetKeyIdsInUse()
		instance.Status.CertificateKeyIds = keyIdsInUse.Certificate
		instance.Status.PasswordKeyIds = keyIdsInUse.Password
		return keyIdsInUse
	}()

	credentialsSet, validCredentials, err := secretsExtractor.GetPreviousCredentialsSet(instance.Status.SynchronizationSecretName)
	if err != nil {
		return nil, fmt.Errorf("extracting credentials set from secret: %w", err)
	}

	return &transactionSecrets.Secrets{
		Credentials: transactionSecrets.Credentials{
			Set:   credentialsSet,
			Valid: validCredentials,
		},
		DataKeys:       dataKeys,
		KeyIdsInUse:    keyIdsInUse,
		ManagedSecrets: *managedSecrets,
	}, nil
}

func (s secretsReconciler) Process(tx transaction.Transaction, applicationResult *result.Application) error {
	// return early if no operations needed
	if tx.Options.Process.Secret.Valid && !tx.Options.Process.Secret.Rotate && applicationResult.IsNotModified() {
		return nil
	}

	var err error

	credentialsSet := tx.Secrets.Credentials.Set
	keyIdsInUse := tx.Secrets.KeyIdsInUse

	switch {
	case !tx.Options.Process.Secret.Valid:
		credentialsSet, keyIdsInUse, err = s.Azure().AddCredentials(tx, keyIdsInUse)
		if err != nil {
			return fmt.Errorf("adding azure credentials: %w", err)
		}
	case tx.Options.Process.Secret.Rotate:
		credentialsSet, keyIdsInUse, err = s.Azure().RotateCredentials(tx, *credentialsSet, keyIdsInUse)
		if err != nil {
			return fmt.Errorf("rotating azure credentials: %w", err)
		}
	}

	if err := s.createOrUpdate(tx, *applicationResult, *credentialsSet); err != nil {
		return err
	}

	if !tx.Options.Process.Secret.Valid || tx.Options.Process.Secret.Rotate {
		tx.Instance.Status.CertificateKeyIds = keyIdsInUse.Certificate
		tx.Instance.Status.PasswordKeyIds = keyIdsInUse.Password
		now := metav1.Now()
		tx.Instance.Status.SynchronizationSecretRotationTime = &now
	}

	return nil
}

func (s secretsReconciler) createOrUpdate(tx transaction.Transaction, result result.Application, set credentials.Set) error {
	secretName := tx.Instance.Spec.SecretName
	objectMeta := kubernetes.ObjectMeta(secretName, tx.Instance.GetNamespace(), labels.Labels(tx.Instance))
	objectMeta.SetAnnotations(map[string]string{
		annotations.StakaterReloaderKey: "true",
	})

	secret := &corev1.Secret{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		Type: corev1.SecretTypeOpaque,
	}

	stringData, err := secrets.SecretData(result, set, s.azureOpenIdConfig, tx.Secrets.DataKeys)
	if err != nil {
		return fmt.Errorf("while creating secret data for secret '%s': %w", secretName, err)
	}

	secretMutateFn := func() error {
		secret.StringData = stringData
		return ctrl.SetControllerReference(tx.Instance, secret, s.scheme)
	}

	res, err := controllerutil.CreateOrUpdate(tx.Ctx, s.client, secret, secretMutateFn)
	if err != nil {
		return fmt.Errorf("creating or updating secret %s: %w", secretName, err)
	}

	tx.Logger.Infof("secret '%s' %s", secretName, res)
	return nil
}

func (s secretsReconciler) getManaged(ctx context.Context, instance *v1.AzureAdApplication) (*kubernetes.SecretLists, error) {
	// fetch all application pods for this app
	podList, err := kubernetes.ListPodsForApplication(ctx, s.reader, instance.GetName(), instance.GetNamespace())
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	var allSecrets corev1.SecretList
	opts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(labels.Labels(instance)),
	}
	if err := s.reader.List(ctx, &allSecrets, opts...); err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := kubernetes.ListUsedAndUnusedSecretsForPods(allSecrets, podList)
	return &podSecrets, nil
}

func (s secretsReconciler) DeleteUnused(tx transaction.Transaction) error {
	unused := tx.Secrets.ManagedSecrets.Unused

	for i, oldSecret := range unused.Items {
		if oldSecret.Name == tx.Instance.Spec.SecretName {
			continue
		}
		tx.Logger.Infof("deleting unused secret '%s'...", oldSecret.Name)
		if err := s.client.Delete(tx.Ctx, &unused.Items[i]); err != nil {
			return fmt.Errorf("deleting unused secret: %w", err)
		}
	}
	return nil
}
