package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	gocache "github.com/patrickmn/go-cache"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Get returns a Graph API Application entity, which represents in Application in AAD
func (c client) Get(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.Application, error) {
	return c.getApplication(ctx, credential)
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
	if _, found := c.applicationsCache.Get(credential.GetUniqueName()); found {
		return found, nil
	}
	applications, err := c.allApplications(ctx, filterByName(credential.GetUniqueName()))
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return len(applications) > 0, nil
}

func (c client) getApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.Application, error) {
	if application, found := c.applicationsCache.Get(credential.GetUniqueName()); found {
		return application.(msgraph.Application), nil
	}
	applications, err := c.allApplications(ctx, filterByName(credential.GetUniqueName()))
	if err != nil {
		return msgraph.Application{}, err
	}
	if len(applications) == 0 {
		return msgraph.Application{}, fmt.Errorf("could not find azure application with name '%s'", credential.GetName())
	}
	if len(applications) > 1 {
		return msgraph.Application{}, fmt.Errorf("found more than one azure application with name '%s'", credential.GetName())
	}
	return applications[0], nil
}

func (c client) allApplications(ctx context.Context, filters ...string) ([]msgraph.Application, error) {
	var applications []msgraph.Application

	r := c.graphClient.Applications().Request()
	r.Filter(mapFiltersToFilter(filters))
	applications, err := r.GetN(ctx, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get list applications: %w", err)
	}
	c.addToCache(applications)
	return applications, nil
}

func (c client) addToCache(applications []msgraph.Application) {
	for _, app := range applications {
		c.applicationsCache.Set(*app.DisplayName, app, gocache.NoExpiration)
	}
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
