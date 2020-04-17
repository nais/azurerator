package client

import (
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure/util"
	gocache "github.com/patrickmn/go-cache"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// ApplicationExists returns an indication of whether the application exists in AAD or not
func (c client) ApplicationExists(credential v1alpha1.AzureAdCredential) (bool, error) {
	exists, err := c.applicationExists(credential)
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return exists, nil
}

func (c client) applicationExists(credential v1alpha1.AzureAdCredential) (bool, error) {
	if len(credential.Status.ObjectId) > 0 && len(credential.Status.ClientId) > 0 {
		return true, nil
	}
	if _, found := c.applicationsCache.Get(credential.Name); found {
		return found, nil
	}
	applications, err := c.allApplications(util.FilterByName(credential.GetName()))
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return len(applications) > 0, nil
}

func (c client) getApplication(credential v1alpha1.AzureAdCredential) (msgraph.Application, error) {
	if application, found := c.applicationsCache.Get(credential.Name); found {
		return application.(msgraph.Application), nil
	}
	applications, err := c.allApplications(util.FilterByName(credential.GetName()))
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

func (c client) allApplications(filters ...string) ([]msgraph.Application, error) {
	var applications []msgraph.Application

	r := c.graphClient.Applications().Request()
	r.Filter(util.MapFiltersToFilter(filters))
	applications, err := r.GetN(c.ctx, 1000)
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
