package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/liberator/pkg/kubernetes"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/labels"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
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
	JwkKey           = "AZURE_APP_JWK"
	PasswordIdKey    = "AZURE_APP_PASSWORD_KEY_ID"
	PreAuthAppsKey   = "AZURE_APP_PRE_AUTHORIZED_APPS"
	TenantId         = "AZURE_APP_TENANT_ID"
	WellKnownUrlKey  = "AZURE_APP_WELL_KNOWN_URL"

	OpenIDConfigIssuerKey        = "AZURE_OPENID_CONFIG_ISSUER"
	OpenIDConfigJwksUriKey       = "AZURE_OPENID_CONFIG_JWKS_URI"
	OpenIDConfigTokenEndpointKey = "AZURE_OPENID_CONFIG_TOKEN_ENDPOINT"
)

var AllKeys = []string{
	CertificateIdKey,
	ClientIdKey,
	ClientSecretKey,
	JwksKey,
	JwkKey,
	PasswordIdKey,
	PreAuthAppsKey,
	TenantId,
	WellKnownUrlKey,
	OpenIDConfigIssuerKey,
	OpenIDConfigJwksUriKey,
	OpenIDConfigTokenEndpointKey,
}

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch
// TODO(tronghn) - refactor
func CreateOrUpdate(
	ctx context.Context,
	instance *v1.AzureAdApplication,
	application azure.ApplicationResult,
	cli client.Client,
	scheme *runtime.Scheme,
	azureOpenIDConfig config.AzureOpenIdConfig,
) (controllerutil.OperationResult, error) {
	spec, err := spec(instance, application, azureOpenIDConfig)
	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("unable to create secretSpec object: %w", err)
	}

	if err := ctrl.SetControllerReference(instance, spec, scheme); err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("failed to set controller reference: %w", err)
	}

	err = cli.Create(ctx, spec)
	res := controllerutil.OperationResultCreated

	if errors.IsAlreadyExists(err) {
		err = cli.Update(ctx, spec)
		res = controllerutil.OperationResultUpdated
	}

	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("unable to apply secretSpec: %w", err)
	}
	return res, nil
}

func GetManaged(ctx context.Context, instance *v1.AzureAdApplication, reader client.Reader) (*kubernetes.SecretLists, error) {
	// fetch all application pods for this app
	podList, err := kubernetes.ListPodsForApplication(ctx, reader, instance.GetName(), instance.GetNamespace())
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	allSecrets, err := getAll(ctx, instance, reader)
	if err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := kubernetes.ListUsedAndUnusedSecretsForPods(allSecrets, podList)
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

func spec(instance *v1.AzureAdApplication, app azure.ApplicationResult, azureOpenIDConfig config.AzureOpenIdConfig) (*corev1.Secret, error) {
	data, err := stringData(app, azureOpenIDConfig)
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

func stringData(app azure.ApplicationResult, azureOpenIDConfig config.AzureOpenIdConfig) (map[string]string, error) {
	jwkJson, err := json.Marshal(app.Certificate.Jwk.Private)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private JWK: %w", err)
	}
	jwksJson, err := json.Marshal(app.Certificate.Jwk.ToPrivateJwks())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private JWKS: %w", err)
	}
	preAuthAppsJson, err := json.Marshal(app.PreAuthorizedApps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal preauthorized apps: %w", err)
	}
	return map[string]string{
		CertificateIdKey:             app.Certificate.KeyId.Latest,
		ClientIdKey:                  app.ClientId,
		ClientSecretKey:              app.Password.ClientSecret,
		JwksKey:                      string(jwksJson),
		JwkKey:                       string(jwkJson),
		PasswordIdKey:                app.Password.KeyId.Latest,
		PreAuthAppsKey:               string(preAuthAppsJson),
		TenantId:                     app.Tenant,
		WellKnownUrlKey:              azureOpenIDConfig.WellKnownEndpoint,
		OpenIDConfigIssuerKey:        azureOpenIDConfig.Issuer,
		OpenIDConfigJwksUriKey:       azureOpenIDConfig.JwksURI,
		OpenIDConfigTokenEndpointKey: azureOpenIDConfig.TokenEndpoint,
	}, nil
}
