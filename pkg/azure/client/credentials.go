package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/keycredential"
	"github.com/nais/azureator/pkg/azure/client/passwordcredential"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/transaction"
	"github.com/nais/msgraph.go/v1.0"
)

type credentialsClient struct {
	Client
}

func (c credentialsClient) KeyCredential() keycredential.KeyCredential {
	return keycredential.NewKeyCredential(c)
}

func (c credentialsClient) PasswordCredential() passwordcredential.PasswordCredential {
	return passwordcredential.NewPasswordCredential(c)
}

// Add adds credentials for an existing AAD application
func (c credentialsClient) Add(tx transaction.Transaction) (credentials.Set, error) {
	if err := credentials.Wait(tx.Ctx, c.DelayIntervalBetweenModifications()); err != nil {
		return credentials.Set{}, err
	}

	currPasswordCredential, err := c.PasswordCredential().Add(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("adding current password credential: %w", err)
	}

	if err := credentials.Wait(tx.Ctx, c.DelayIntervalBetweenModifications()); err != nil {
		return credentials.Set{}, err
	}

	nextPasswordCredential, err := c.PasswordCredential().Add(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("adding next password credential: %w", err)
	}

	if err := credentials.Wait(tx.Ctx, c.DelayIntervalBetweenModifications()); err != nil {
		return credentials.Set{}, err
	}

	keyCredentialSet, err := c.KeyCredential().Add(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("adding key credential set: %w", err)
	}

	return credentials.Set{
		Current: credentials.Credentials{
			Certificate: credentials.Certificate{
				KeyId: string(*keyCredentialSet.Current.KeyCredential.KeyID),
				Jwk:   keyCredentialSet.Current.Jwk,
			},
			Password: credentials.Password{
				KeyId:        string(*currPasswordCredential.KeyID),
				ClientSecret: *currPasswordCredential.SecretText,
			},
		},
		Next: credentials.Credentials{
			Certificate: credentials.Certificate{
				KeyId: string(*keyCredentialSet.Next.KeyCredential.KeyID),
				Jwk:   keyCredentialSet.Next.Jwk,
			},
			Password: credentials.Password{
				KeyId:        string(*nextPasswordCredential.KeyID),
				ClientSecret: *nextPasswordCredential.SecretText,
			},
		},
	}, nil
}

// DeleteExpired deletes all expired credentials for the application in Azure AD.
func (c credentialsClient) DeleteExpired(tx transaction.Transaction) error {
	err := c.KeyCredential().DeleteExpired(tx)
	if err != nil {
		return fmt.Errorf("deleting expired key credentials: %w", err)
	}

	err = c.PasswordCredential().DeleteExpired(tx)
	if err != nil {
		return fmt.Errorf("deleting expired password credentials: %w", err)
	}

	return nil
}

// DeleteUnused deletes unused credentials for an existing AAD application.
func (c credentialsClient) DeleteUnused(tx transaction.Transaction) error {
	err := c.KeyCredential().DeleteUnused(tx)
	if err != nil {
		return fmt.Errorf("deleting unused key credentials: %w", err)
	}

	err = c.PasswordCredential().DeleteUnused(tx)
	if err != nil {
		return fmt.Errorf("deleting unused password credentials: %w", err)
	}

	return nil
}

// Purge removes all credentials for the application in Azure AD.
func (c credentialsClient) Purge(tx transaction.Transaction) error {
	err := c.PasswordCredential().Purge(tx)
	if err != nil {
		return fmt.Errorf("purging password credentials: %w", err)
	}

	err = c.KeyCredential().Purge(tx)
	if err != nil {
		return fmt.Errorf("purging key credentials: %w", err)
	}

	return nil
}

// Rotate rotates credentials for an existing AAD application
func (c credentialsClient) Rotate(tx transaction.Transaction) (credentials.Set, error) {
	if err := credentials.Wait(tx.Ctx, c.DelayIntervalBetweenModifications()); err != nil {
		return credentials.Set{}, err
	}

	nextPasswordCredential, err := c.PasswordCredential().Rotate(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("rotating password credential: %w", err)
	}

	if err := credentials.Wait(tx.Ctx, c.DelayIntervalBetweenModifications()); err != nil {
		return credentials.Set{}, err
	}

	nextKeyCredential, nextJwk, err := c.KeyCredential().Rotate(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("rotating key credential: %w", err)
	}

	return credentials.Set{
		Current: tx.Secrets.LatestCredentials.Set.Next,
		Next: credentials.Credentials{
			Certificate: credentials.Certificate{
				KeyId: string(*nextKeyCredential.KeyID),
				Jwk:   *nextJwk,
			},
			Password: credentials.Password{
				KeyId:        string(*nextPasswordCredential.KeyID),
				ClientSecret: *nextPasswordCredential.SecretText,
			},
		},
	}, nil
}

// Validate validates the given credentials set against the actual state for the application in Azure AD.
func (c credentialsClient) Validate(tx transaction.Transaction, existing credentials.Set) (bool, error) {
	app, err := c.Application().Get(tx)
	if err != nil {
		return false, fmt.Errorf("validating credentials: %w", err)
	}
	problems := missingCredentials(app, existing, time.Now())
	if len(problems) > 0 {
		tx.Logger.Warnf("credential validation failed: %s (password key IDs: current=%s, next=%s; certificate key IDs: current=%s, next=%s)",
			strings.Join(problems, "; "), existing.Current.Password.KeyId, existing.Next.Password.KeyId,
			existing.Current.Certificate.KeyId, existing.Next.Certificate.KeyId)
		return false, nil
	}
	return true, nil
}

func missingCredentials(app msgraph.Application, expected credentials.Set, now time.Time) []string {
	passwords := make(map[string]*time.Time, len(app.PasswordCredentials))
	for _, actual := range app.PasswordCredentials {
		if actual.KeyID != nil {
			passwords[string(*actual.KeyID)] = actual.EndDateTime
		}
	}
	certificates := make(map[string]*time.Time, len(app.KeyCredentials))
	for _, actual := range app.KeyCredentials {
		if actual.KeyID != nil {
			certificates[string(*actual.KeyID)] = actual.EndDateTime
		}
	}

	var problems []string
	for _, problem := range []string{
		checkCredential("current password", expected.Current.Password.KeyId, passwords, now),
		checkCredential("current certificate", expected.Current.Certificate.KeyId, certificates, now),
		checkCredential("next password", expected.Next.Password.KeyId, passwords, now),
		checkCredential("next certificate", expected.Next.Certificate.KeyId, certificates, now),
	} {
		if problem != "" {
			problems = append(problems, problem)
		}
	}
	return problems
}

// checkCredential returns a description of why the credential is invalid, or "" if it is valid.
func checkCredential(name, id string, expiries map[string]*time.Time, now time.Time) string {
	expiry, found := expiries[id]
	switch {
	case id == "" || !found:
		return fmt.Sprintf("%s credential %q is missing", name, id)
	case expiry == nil:
		return fmt.Sprintf("%s credential %q is expired (missing expiry)", name, id)
	case !expiry.After(now):
		return fmt.Sprintf("%s credential %q is expired", name, id)
	}
	return ""
}

func NewCredentials(client Client) azure.Credentials {
	return credentialsClient{Client: client}
}
