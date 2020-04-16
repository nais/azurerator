package client

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
)

// UpdateApplication updates an existing AAD application
func (c client) UpdateApplication(credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return c.updateApplication(credential)
}

// TODO
func (c client) updateApplication(credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return azure.Application{}, nil
}

// TODO
func (c client) addClientSecret(credential v1alpha1.AzureAdCredential) {
	_, _ = c.applicationsClient.UpdatePasswordCredentials(c.ctx, "", graphrbac.PasswordCredentialsUpdateParameters{
		Value: &[]graphrbac.PasswordCredential{
			{
				StartDate: &date.Time{Time: time.Now()},
				EndDate:   &date.Time{Time: time.Now().AddDate(0, 0, 1)},
				KeyID:     to.StringPtr("mykeyid"),
				Value:     to.StringPtr("mypassword"),
			},
		},
	})
}
