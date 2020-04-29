package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Get returns a Graph API Application entity, which represents in Application in AAD
func (c client) Get(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.Application, error) {
	if len(credential.Status.ObjectId) == 0 {
		return c.getApplicationByName(ctx, credential)
	}
	return c.getApplicationById(ctx, credential)
}

// Exists returns an indication of whether the application exists in AAD or not
func (c client) Exists(ctx context.Context, credential v1alpha1.AzureAdCredential) (bool, error) {
	exists, err := c.applicationExists(ctx, credential)
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return exists, nil
}

func (c client) applicationExists(ctx context.Context, credential v1alpha1.AzureAdCredential) (bool, error) {
	applications, err := c.allApplications(ctx, filterByName(credential.GetUniqueName()))
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return len(applications) > 0, nil
}

func (c client) getApplicationById(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.Application, error) {
	objectId := credential.Status.ObjectId
	application, err := c.graphClient.Applications().ID(objectId).Request().Get(ctx)
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("failed to lookup azure application with ID '%s'", objectId)
	}
	return *application, nil
}

func (c client) getApplicationByName(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.Application, error) {
	name := credential.GetUniqueName()
	applications, err := c.allApplications(ctx, filterByName(name))
	if err != nil {
		return msgraph.Application{}, err
	}
	if len(applications) == 0 {
		return msgraph.Application{}, fmt.Errorf("could not find azure application with name '%s'", name)
	}
	if len(applications) > 1 {
		return msgraph.Application{}, fmt.Errorf("found more than one azure application with name '%s'", name)
	}
	return applications[0], nil
}

func (c client) getExistingKeyCredential(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.KeyCredential, error) {
	application, err := c.Get(ctx, credential)
	if err != nil {
		return msgraph.KeyCredential{}, err
	}
	for _, keyCredential := range application.KeyCredentials {
		if string(*keyCredential.KeyID) == credential.Status.CertificateKeyId {
			return keyCredential, nil
		}
	}
	return msgraph.KeyCredential{}, fmt.Errorf("failed to find previous key ID in Status field")
}

func (c client) allApplications(ctx context.Context, filters ...string) ([]msgraph.Application, error) {
	var applications []msgraph.Application

	r := c.graphClient.Applications().Request()
	r.Filter(mapFiltersToFilter(filters))
	applications, err := r.GetN(ctx, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get list applications: %w", err)
	}
	return applications, nil
}

func mapFiltersToFilter(filters []string) string {
	if len(filters) > 0 {
		return strings.Join(filters[:], " ")
	} else {
		return ""
	}
}

func filterByName(name string) string {
	return fmt.Sprintf("displayName eq '%s'", name)
}
