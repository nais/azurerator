package client

import (
	"testing"
	"time"

	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/credentials"
)

func TestMissingCredentials(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	expiry := now.Add(time.Hour)
	shortExpiry := expiry.Add(-2 * time.Second)
	expected := credentials.Set{
		Current: credentials.Credentials{
			Password:    credentials.Password{KeyId: "password-current"},
			Certificate: credentials.Certificate{KeyId: "certificate-current"},
		},
		Next: credentials.Credentials{
			Password:    credentials.Password{KeyId: "password-next"},
			Certificate: credentials.Certificate{KeyId: "certificate-next"},
		},
	}
	validApp := msgraph.Application{
		PasswordCredentials: []msgraph.PasswordCredential{
			{KeyID: uuid("password-current"), EndDateTime: &expiry},
			{KeyID: uuid("password-next"), EndDateTime: &expiry},
		},
		KeyCredentials: []msgraph.KeyCredential{
			{KeyID: uuid("certificate-current"), EndDateTime: &expiry},
			{KeyID: uuid("certificate-next"), EndDateTime: &expiry},
		},
	}
	tests := []struct {
		name          string
		app           msgraph.Application
		minimumExpiry time.Time
		want          []string
	}{
		{name: "valid", app: validApp},
		{name: "missing password", app: msgraph.Application{
			KeyCredentials: validApp.KeyCredentials,
		}, want: []string{
			`current password credential "password-current" is missing`,
			`next password credential "password-next" is missing`,
		}},
		{name: "missing certificate", app: msgraph.Application{
			PasswordCredentials: validApp.PasswordCredentials,
		}, want: []string{
			`current certificate credential "certificate-current" is missing`,
			`next certificate credential "certificate-next" is missing`,
		}},
		{name: "expired", app: withExpiry(validApp, &now), minimumExpiry: expiry, want: []string{
			`current password credential "password-current" is expired (expired at 2025-01-02T03:04:05Z)`,
			`current certificate credential "certificate-current" is expired (expired at 2025-01-02T03:04:05Z)`,
			`next password credential "password-next" is expired (expired at 2025-01-02T03:04:05Z)`,
			`next certificate credential "certificate-next" is expired (expired at 2025-01-02T03:04:05Z)`,
		}},
		{name: "nil expiry", app: withExpiry(validApp, nil), want: []string{
			`current password credential "password-current" is expired (missing expiry)`,
			`current certificate credential "certificate-current" is expired (missing expiry)`,
			`next password credential "password-next" is expired (missing expiry)`,
			`next certificate credential "certificate-next" is expired (missing expiry)`,
		}},
		{name: "short next password", app: msgraph.Application{
			PasswordCredentials: []msgraph.PasswordCredential{
				{KeyID: uuid("password-current"), EndDateTime: &expiry},
				{KeyID: uuid("password-next"), EndDateTime: &shortExpiry},
			},
			KeyCredentials: validApp.KeyCredentials,
		}, minimumExpiry: expiry.Add(-time.Second), want: []string{
			`next password credential "password-next" expires before required rotation window (expires at 2025-01-02T04:04:03Z, must expire after 2025-01-02T04:04:04Z)`,
		}},
		{name: "short next certificate", app: msgraph.Application{
			PasswordCredentials: validApp.PasswordCredentials,
			KeyCredentials: []msgraph.KeyCredential{
				{KeyID: uuid("certificate-current"), EndDateTime: &expiry},
				{KeyID: uuid("certificate-next"), EndDateTime: &shortExpiry},
			},
		}, minimumExpiry: expiry.Add(-time.Second), want: []string{
			`next certificate credential "certificate-next" expires before required rotation window (expires at 2025-01-02T04:04:03Z, must expire after 2025-01-02T04:04:04Z)`,
		}},
		{name: "next credentials meet rotation window", app: validApp, minimumExpiry: expiry.Add(-time.Second)},
		{name: "next credentials expire at rotation boundary", app: validApp, minimumExpiry: expiry, want: []string{
			`next password credential "password-next" expires before required rotation window (expires at 2025-01-02T04:04:05Z, must expire after 2025-01-02T04:04:05Z)`,
			`next certificate credential "certificate-next" expires before required rotation window (expires at 2025-01-02T04:04:05Z, must expire after 2025-01-02T04:04:05Z)`,
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, missingCredentials(test.app, expected, now, test.minimumExpiry))
		})
	}
}

func uuid(id string) *msgraph.UUID {
	value := msgraph.UUID(id)
	return &value
}

func withExpiry(app msgraph.Application, expiry *time.Time) msgraph.Application {
	app.PasswordCredentials = append([]msgraph.PasswordCredential(nil), app.PasswordCredentials...)
	app.KeyCredentials = append([]msgraph.KeyCredential(nil), app.KeyCredentials...)
	for i := range app.PasswordCredentials {
		app.PasswordCredentials[i].EndDateTime = expiry
	}
	for i := range app.KeyCredentials {
		app.KeyCredentials[i].EndDateTime = expiry
	}
	return app
}
