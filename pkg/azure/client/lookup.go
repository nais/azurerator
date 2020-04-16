package client

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure/util"
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
	if len(credential.Status.ObjectId) > 0 || len(credential.Status.ClientId) > 0 {
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

func (c client) allApplications(filters ...string) ([]graphrbac.Application, error) {
	var applications []graphrbac.Application
	var result graphrbac.ApplicationListResultPage

	result, err := c.applicationsClient.List(c.ctx, util.MapFiltersToFilter(filters))
	if err != nil {
		return nil, fmt.Errorf("failed to get list applications: %w", err)
	}
	for {
		applications = append(applications, result.Values()...)
		err = result.NextWithContext(c.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get list applications: %w", err)
		}
		if !result.NotDone() {
			c.addToCache(applications)
			return applications, nil
		}
	}
}
