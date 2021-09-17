package customresources_test

import (
	"strconv"
	"testing"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/event"
)

func TestAzureAdApplication_IsHashChanged(t *testing.T) {
	t.Run("Application with unchanged spec should be synchronized", func(t *testing.T) {
		app := minimalApplication()
		actual, err := customresources.IsHashChanged(app)
		assert.NoError(t, err)
		assert.False(t, actual)
	})
	t.Run("Application with changed spec should not be synchronized", func(t *testing.T) {
		app := minimalApplication()
		app.Spec.LogoutUrl = "yolo"
		actual, err := customresources.IsHashChanged(app)
		assert.NoError(t, err)
		assert.True(t, actual)
	})
}

func TestHasExtraPolicy(t *testing.T) {
	claims := &nais_io_v1.AzureAdClaims{
		Extra: []nais_io_v1.AzureAdExtraClaim{
			"some-policy",
			"some-other-policy",
		},
	}

	t.Run("Checking for non-existent extra claim should return false", func(t *testing.T) {
		actual := customresources.HasExtraPolicy(claims, "non-existent")
		assert.False(t, actual)
	})
	t.Run("Checking for extra claim should return true", func(t *testing.T) {
		actual := customresources.HasExtraPolicy(claims, "some-policy")
		assert.True(t, actual)
	})
}

func TestIsSecretNameChanged(t *testing.T) {
	t.Run("Application with unchanged secret name", func(t *testing.T) {
		app := minimalApplication()
		shouldUpdate := customresources.SecretNameChanged(app)
		assert.False(t, shouldUpdate)
	})

	t.Run("Application with changed secret name", func(t *testing.T) {
		app := minimalApplication()
		app.Spec.SecretName = "some-secret"
		shouldUpdate := customresources.SecretNameChanged(app)
		assert.True(t, shouldUpdate)
	})

	t.Run("Application with not set synchronized secret name in status", func(t *testing.T) {
		app := minimalApplication()
		app.Status.SynchronizationSecretName = ""
		shouldUpdate := customresources.SecretNameChanged(app)
		assert.True(t, shouldUpdate)
	})
}

func TestHasExpiredSecrets(t *testing.T) {
	t.Run("not set rotation time should return not expired", func(t *testing.T) {
		app := minimalApplication()
		app.Status.SynchronizationSecretRotationTime = nil

		shouldUpdate := customresources.HasExpiredSecrets(app, time.Minute)
		assert.False(t, shouldUpdate)
	})

	t.Run("valid secret should return not expired", func(t *testing.T) {
		app := minimalApplication()
		shouldUpdate := customresources.HasExpiredSecrets(app, time.Minute)

		assert.False(t, shouldUpdate)
	})

	t.Run("expired secret should return expired", func(t *testing.T) {
		app := minimalApplication()

		expiredTime := metav1.NewTime(metav1.Now().Add(-1 * time.Minute))
		app.Status.SynchronizationSecretRotationTime = &expiredTime

		shouldUpdate := customresources.HasExpiredSecrets(app, time.Minute)
		assert.True(t, shouldUpdate)
	})
}

func TestHasResynchronizeAnnotation(t *testing.T) {
	t.Run("not set annotation should not resynchronize", func(t *testing.T) {
		app := minimalApplication()

		hasAnnotation := customresources.HasResynchronizeAnnotation(app)
		assert.False(t, hasAnnotation)
	})

	t.Run("set annotation should synchronize regardless of value", func(t *testing.T) {
		app := minimalApplication()
		annotations.SetAnnotation(app, annotations.ResynchronizeKey, strconv.FormatBool(false))

		hasAnnotation := customresources.HasResynchronizeAnnotation(app)
		assert.True(t, hasAnnotation)

		app = minimalApplication()
		annotations.SetAnnotation(app, annotations.ResynchronizeKey, strconv.FormatBool(true))

		hasAnnotation = customresources.HasResynchronizeAnnotation(app)
		assert.True(t, hasAnnotation)
	})
}

func TestHasRotateAnnotation(t *testing.T) {
	t.Run("not set annotation should not rotate", func(t *testing.T) {
		app := minimalApplication()

		hasAnnotation := customresources.HasRotateAnnotation(app)
		assert.False(t, hasAnnotation)
	})

	t.Run("set annotation should rotate regardless of value", func(t *testing.T) {
		app := minimalApplication()
		annotations.SetAnnotation(app, annotations.RotateKey, strconv.FormatBool(false))

		hasAnnotation := customresources.HasRotateAnnotation(app)
		assert.True(t, hasAnnotation)

		app = minimalApplication()
		annotations.SetAnnotation(app, annotations.RotateKey, strconv.FormatBool(true))

		hasAnnotation = customresources.HasRotateAnnotation(app)
		assert.True(t, hasAnnotation)
	})
}

func TestHasMatchingPreAuthorizedApp(t *testing.T) {
	e := event.NewEvent("1", event.Created, &metav1.ObjectMeta{
		Name:        "some-app",
		Namespace:   "test-namespace",
		ClusterName: "test-cluster",
	})

	for _, test := range []struct {
		name     string
		rule     nais_io_v1.AccessPolicyRule
		expected bool
	}{
		{
			name:     "no rule",
			rule:     nais_io_v1.AccessPolicyRule{},
			expected: false,
		},
		{
			name: "non-matching app",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "another-app",
			},
			expected: false,
		},
		{
			name: "non-matching namespace",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Namespace:   "another-namespace",
			},
			expected: false,
		},
		{
			name: "non-matching cluster",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Cluster:     "another-cluster",
			},
			expected: false,
		},
		{
			name: "non-matching namespace and cluster",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Namespace:   "another-namespace",
				Cluster:     "another-cluster",
			},
			expected: false,
		},
		{
			name: "no matching fields",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "another-app",
				Namespace:   "another-namespace",
				Cluster:     "another-cluster",
			},
			expected: false,
		},
		{
			name: "all fields matching",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Namespace:   "test-namespace",
				Cluster:     "test-cluster",
			},
			expected: true,
		},
		{
			name: "matching app and namespace, omitted cluster",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Namespace:   "test-namespace",
			},
			expected: true,
		},
		{
			name: "matching app and cluster, omitted namespace",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Cluster:     "test-cluster",
			},
			expected: true,
		},
		{
			name: "matching app, omitted cluster and namespace",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
			},
			expected: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := minimalApplication()
			app.Spec.PreAuthorizedApplications = []nais_io_v1.AccessPolicyInboundRule{{AccessPolicyRule: test.rule}}

			actual := customresources.HasMatchingPreAuthorizedApp(*app, e)

			if test.expected {
				assert.True(t, actual)
			} else {
				assert.False(t, actual)
			}
		})
	}
}

func minimalApplication() *nais_io_v1.AzureAdApplication {
	now := metav1.Now()
	return &nais_io_v1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: nais_io_v1.AzureAdApplicationSpec{
			ReplyUrls:                 nil,
			PreAuthorizedApplications: nil,
			LogoutUrl:                 "test",
			SecretName:                "test",
		},
		Status: nais_io_v1.AzureAdApplicationStatus{
			PasswordKeyIds:                    []string{"test"},
			CertificateKeyIds:                 []string{"test"},
			ClientId:                          "test",
			ObjectId:                          "test",
			ServicePrincipalId:                "test",
			SynchronizationHash:               "b85f1aaff45fcfc2",
			SynchronizationSecretName:         "test",
			SynchronizationSecretRotationTime: &now,
		},
	}
}
