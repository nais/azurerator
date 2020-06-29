package secrets

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/azure"
	azureConfig "github.com/nais/azureator/pkg/azure/config"
	"github.com/nais/azureator/pkg/labels"
	"github.com/nais/azureator/pkg/pods"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	CertificateIdKey = "AZURE_APP_CERTIFICATE_KEY_ID"
	ClientIdKey      = "AZURE_APP_CLIENT_ID"
	ClientSecretKey  = "AZURE_APP_CLIENT_SECRET"
	JwksKey          = "AZURE_APP_JWKS"
	PasswordIdKey    = "AZURE_APP_PASSWORD_KEY_ID"
	PreAuthAppsKey   = "AZURE_APP_PRE_AUTHORIZED_APPS"
	WellKnownUrlKey  = "AZURE_APP_WELL_KNOWN_URL"
)

var AllKeys = []string{
	CertificateIdKey,
	ClientIdKey,
	ClientSecretKey,
	JwksKey,
	PasswordIdKey,
	PreAuthAppsKey,
	WellKnownUrlKey,
}

func CreateOrUpdate(ctx context.Context, instance *v1.AzureAdApplication, application azure.Application, cli client.Client, scheme *runtime.Scheme) (controllerutil.OperationResult, error) {
	spec, err := spec(instance, application)
	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("unable to create secretSpec object: %s", err)
	}

	if err := ctrl.SetControllerReference(instance, spec, scheme); err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("failed to set controller reference %w", err)
	}

	err = cli.Create(ctx, spec)
	res := controllerutil.OperationResultCreated

	if errors.IsAlreadyExists(err) {
		err = cli.Update(ctx, spec)
		res = controllerutil.OperationResultUpdated
	}

	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("unable to apply secretSpec: %s", err)
	}
	return res, nil
}

func GetManaged(ctx context.Context, instance *v1.AzureAdApplication, reader client.Reader) (*Lists, error) {
	// fetch all application pods for this app
	podList, err := pods.GetForApplication(ctx, instance, reader)
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	allSecrets, err := getAll(ctx, instance, reader)
	if err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := podSecretLists(allSecrets, *podList)
	return &podSecrets, nil
}

func Delete(ctx context.Context, secret corev1.Secret, cli client.Client) error {
	if err := cli.Delete(ctx, &secret); err != nil {
		return fmt.Errorf("failed to delete unused secret: %w", err)
	}
	return nil
}

func getAll(ctx context.Context, instance *v1.AzureAdApplication, reader client.Reader) (corev1.SecretList, error) {
	var list corev1.SecretList
	mLabels := client.MatchingLabels{
		labels.AppLabelKey:  instance.GetName(),
		labels.TypeLabelKey: labels.TypeLabelValue,
	}
	if err := reader.List(ctx, &list, client.InNamespace(instance.Namespace), mLabels); err != nil {
		return list, err
	}
	return list, nil
}

func spec(instance *v1.AzureAdApplication, app azure.Application) (*corev1.Secret, error) {
	data, err := stringData(app)
	if err != nil {
		return nil, err
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: objectMeta(instance),
		StringData: data,
		Type:       corev1.SecretTypeOpaque,
	}, nil
}

func objectMeta(instance *v1.AzureAdApplication) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      instance.Spec.SecretName,
		Namespace: instance.Namespace,
		Labels:    labels.Labels(instance),
	}
}

func stringData(app azure.Application) (map[string]string, error) {
	jwkPrivateJson, err := json.Marshal(app.Certificate.Jwks.Private)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private JWK: %w", err)
	}
	preAuthAppsJson, err := json.Marshal(app.PreAuthorizedApps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal preauthorized apps: %w", err)
	}
	return map[string]string{
		CertificateIdKey: app.Certificate.KeyId.Latest,
		ClientIdKey:      app.ClientId,
		ClientSecretKey:  app.Password.ClientSecret,
		JwksKey:          string(jwkPrivateJson),
		PasswordIdKey:    app.Password.KeyId.Latest,
		PreAuthAppsKey:   string(preAuthAppsJson),
		WellKnownUrlKey:  azureConfig.WellKnownUrl(app.Tenant),
	}, nil
}
