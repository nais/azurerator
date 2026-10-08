package customresources_test

import (
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/fixtures"
)

func TestIsHashChanged(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*nais_io_v1.AzureAdApplication)
		want   bool
	}{
		{
			name:   "unchanged spec",
			mutate: func(_ *nais_io_v1.AzureAdApplication) {},
			want:   false,
		},
		{
			name: "changed spec",
			mutate: func(app *nais_io_v1.AzureAdApplication) {
				app.Spec.LogoutUrl = "yolo"
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fixtures.MinimalApplication()
			tt.mutate(app)
			actual, err := customresources.IsHashChanged(app)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, actual)
		})
	}
}

func TestIsSecretNameChanged(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*nais_io_v1.AzureAdApplication)
		want   bool
	}{
		{
			name:   "unchanged secret name",
			mutate: func(_ *nais_io_v1.AzureAdApplication) {},
			want:   false,
		},
		{
			name: "changed secret name",
			mutate: func(app *nais_io_v1.AzureAdApplication) {
				app.Spec.SecretName = "some-secret"
			},
			want: true,
		},
		{
			name: "empty synchronized secret name in status",
			mutate: func(app *nais_io_v1.AzureAdApplication) {
				app.Status.SynchronizationSecretName = ""
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fixtures.MinimalApplication()
			tt.mutate(app)
			assert.Equal(t, tt.want, customresources.SecretNameChanged(app))
		})
	}
}

func TestIsSecretRotationDue(t *testing.T) {
	maxAge := 10 * time.Minute

	tests := []struct {
		name   string
		mutate func(*nais_io_v1.AzureAdApplication)
		sleep  time.Duration
		want   bool
	}{
		{
			name: "nil rotation time returns false",
			mutate: func(app *nais_io_v1.AzureAdApplication) {
				app.Status.SynchronizationSecretRotationTime = nil
			},
			want: false,
		},
		{
			name: "just rotated returns false",
			mutate: func(app *nais_io_v1.AzureAdApplication) {
				app.Status.SynchronizationSecretRotationTime = new(metav1.NewTime(time.Now()))
			},
			want: false,
		},
		{
			name: "before expiry returns false",
			mutate: func(app *nais_io_v1.AzureAdApplication) {
				app.Status.SynchronizationSecretRotationTime = new(metav1.NewTime(time.Now()))
			},
			sleep: maxAge - 1*time.Second,
			want:  false,
		},
		{
			name: "at expiry returns true",
			mutate: func(app *nais_io_v1.AzureAdApplication) {
				app.Status.SynchronizationSecretRotationTime = new(metav1.NewTime(time.Now()))
			},
			sleep: maxAge,
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				app := fixtures.MinimalApplication()
				tt.mutate(app)
				time.Sleep(tt.sleep)
				assert.Equal(t, tt.want, customresources.IsSecretRotationDue(app, maxAge))
			})
		})
	}
}

func TestSecretRotationCheckDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Status.SynchronizationSecretRotationTime = new(metav1.Now())
		time.Sleep(10 * time.Minute)
		assert.Equal(t, 50*time.Minute, customresources.SecretRotationCheckDelay(app, time.Hour))
	})
}

func TestCredentialValidationDelay(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name         string
		rotationTime *metav1.Time
		inGrace      bool
	}{
		{name: "nil rotation time"},
		{name: "inside grace period", rotationTime: new(metav1.NewTime(now.Add(-time.Minute))), inGrace: true},
		{name: "outside grace period", rotationTime: new(metav1.NewTime(now.Add(-6 * time.Minute)))},
		{name: "future rotation time", rotationTime: new(metav1.NewTime(now.Add(time.Minute)))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &nais_io_v1.AzureAdApplication{}
			app.Status.SynchronizationSecretRotationTime = tt.rotationTime
			remaining := customresources.CredentialValidationDelay(app, 5*time.Minute)
			if tt.inGrace {
				assert.Greater(t, remaining, 3*time.Minute)
				assert.LessOrEqual(t, remaining, 4*time.Minute)
			} else {
				assert.Zero(t, remaining)
			}
		})
	}
}

func TestAnnotationChecks(t *testing.T) {
	checks := []struct {
		name  string
		key   string
		check func(*nais_io_v1.AzureAdApplication) bool
	}{
		{"HasResynchronizeAnnotation", annotations.ResynchronizeKey, customresources.HasResynchronizeAnnotation},
		{"HasRotateAnnotation", annotations.RotateKey, customresources.HasRotateAnnotation},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			tests := []struct {
				name   string
				mutate func(*nais_io_v1.AzureAdApplication)
				want   bool
			}{
				{
					name:   "not set",
					mutate: func(_ *nais_io_v1.AzureAdApplication) {},
					want:   false,
				},
				{
					name: "set to false",
					mutate: func(app *nais_io_v1.AzureAdApplication) {
						annotations.SetAnnotation(app, c.key, strconv.FormatBool(false))
					},
					want: true,
				},
				{
					name: "set to true",
					mutate: func(app *nais_io_v1.AzureAdApplication) {
						annotations.SetAnnotation(app, c.key, strconv.FormatBool(true))
					},
					want: true,
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					app := fixtures.MinimalApplication()
					tt.mutate(app)
					assert.Equal(t, tt.want, c.check(app))
				})
			}
		})
	}
}
