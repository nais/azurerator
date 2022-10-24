package client

import (
	"fmt"
	"time"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/keycredential"
	"github.com/nais/azureator/pkg/azure/client/passwordcredential"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/transaction"
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
	// sleep to prevent concurrent modification error from Microsoft
	time.Sleep(c.DelayIntervalBetweenModifications())

	currPasswordCredential, err := c.PasswordCredential().Add(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("adding current password credential: %w", err)
	}

	time.Sleep(c.DelayIntervalBetweenModifications())

	nextPasswordCredential, err := c.PasswordCredential().Add(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("adding next password credential: %w", err)
	}

	time.Sleep(c.DelayIntervalBetweenModifications())

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
	time.Sleep(c.DelayIntervalBetweenModifications()) // sleep to prevent concurrent modification error from Microsoft

	nextPasswordCredential, err := c.PasswordCredential().Rotate(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("rotating password credential: %w", err)
	}

	time.Sleep(c.DelayIntervalBetweenModifications())

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
	validPasswordCredentials, err := c.PasswordCredential().Validate(tx, existing)
	if err != nil {
		return false, fmt.Errorf("validating password credentials: %w", err)
	}

	validateKeyCredentials, err := c.KeyCredential().Validate(tx, existing)
	if err != nil {
		return false, fmt.Errorf("validating key credentials: %w", err)
	}

	return validPasswordCredentials && validateKeyCredentials, nil
}

func NewCredentials(client Client) azure.Credentials {
	return credentialsClient{Client: client}
}
