package azureadapplication

import (
	"fmt"
	"github.com/nais/azureator/pkg/metrics"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/nais/azureator/pkg/azure"
)

type azureReconciler struct {
	*Reconciler
}

func (r *Reconciler) azure() azureReconciler {
	return azureReconciler{r}
}

func (a azureReconciler) createOrUpdate(tx transaction) (*azure.ApplicationResult, error) {
	var applicationResult *azure.ApplicationResult

	exists, err := a.exists(tx)
	if err != nil {
		return nil, fmt.Errorf("looking up existence of application: %w", err)
	}

	if !exists {
		applicationResult, err = a.create(tx)
	} else if tx.options.Process.Azure.Synchronize {
		applicationResult, err = a.update(tx)
	} else {
		applicationResult, err = a.notModified(tx)
	}
	if err != nil {
		return nil, err
	}

	a.reportInvalid(tx, applicationResult.PreAuthorizedApps)

	return applicationResult, nil
}

func (a azureReconciler) create(tx transaction) (*azure.ApplicationResult, error) {
	logger.Info("Azure application not found, registering...")

	applicationResult, err := a.AzureClient.Create(tx.toAzureTx())
	if err != nil {
		return nil, fmt.Errorf("creating azure application: %w", err)
	}

	metrics.IncWithNamespaceLabel(metrics.AzureAppsCreatedCount, tx.instance.Namespace)
	a.reportEvent(tx, corev1.EventTypeNormal, v1.EventCreatedInAzure, "Azure application is created")

	tx.instance.Status.ClientId = applicationResult.ClientId
	tx.instance.Status.ObjectId = applicationResult.ObjectId
	tx.instance.Status.ServicePrincipalId = applicationResult.ServicePrincipalId

	return applicationResult, nil
}

func (a azureReconciler) update(tx transaction) (*azure.ApplicationResult, error) {
	logger.Info("Azure application already exists, updating...")

	applicationResult, err := a.AzureClient.Update(tx.toAzureTx())
	if err != nil {
		return nil, fmt.Errorf("updating azure application: %w", err)
	}

	metrics.IncWithNamespaceLabel(metrics.AzureAppsUpdatedCount, tx.instance.Namespace)
	a.reportEvent(tx, corev1.EventTypeNormal, v1.EventUpdatedInAzure, "Azure application is updated")

	return applicationResult, nil
}

func (a azureReconciler) notModified(tx transaction) (*azure.ApplicationResult, error) {
	apps, err := a.AzureClient.GetPreAuthorizedApps(tx.toAzureTx())
	if err != nil {
		return nil, fmt.Errorf("fetching pre-authorized apps: %w", err)
	}
	return &azure.ApplicationResult{
		ClientId:           tx.instance.Status.ClientId,
		ObjectId:           tx.instance.Status.ObjectId,
		ServicePrincipalId: tx.instance.Status.ServicePrincipalId,
		PreAuthorizedApps:  *apps,
		Tenant:             a.Config.Azure.Tenant.Id,
		Result:             azure.OperationResultNotModified,
	}, nil
}

func (a azureReconciler) addCredentials(tx transaction, keyIdsInUse azure.KeyIdsInUse) (*azure.CredentialsSet, azure.KeyIdsInUse, error) {
	logger.Info("adding credentials for Azure application...")

	credentialsSet, err := a.AzureClient.AddCredentials(tx.toAzureTx())
	if err != nil {
		return nil, azure.KeyIdsInUse{}, err
	}

	logger.Info("successfully added credentials for Azure application")

	keyIdsInUse.Certificate = append(keyIdsInUse.Certificate, credentialsSet.Current.Certificate.KeyId)
	keyIdsInUse.Password = append(keyIdsInUse.Password, credentialsSet.Current.Password.KeyId)
	return &credentialsSet, keyIdsInUse, nil
}

func (a azureReconciler) rotateCredentials(tx transaction, existing azure.CredentialsSet, keyIdsInUse azure.KeyIdsInUse) (*azure.CredentialsSet, azure.KeyIdsInUse, error) {
	logger.Info("rotating credentials for Azure application...")

	credentialsSet, err := a.AzureClient.RotateCredentials(tx.toAzureTx(), existing, keyIdsInUse)
	if err != nil {
		return nil, azure.KeyIdsInUse{}, err
	}

	logger.Info("successfully rotated credentials for Azure application")

	keyIdsInUse.Certificate = append(keyIdsInUse.Certificate, credentialsSet.Current.Certificate.KeyId)
	keyIdsInUse.Password = append(keyIdsInUse.Password, credentialsSet.Current.Password.KeyId)

	metrics.IncWithNamespaceLabel(metrics.AzureAppsRotatedCount, tx.instance.Namespace)
	a.reportEvent(tx, corev1.EventTypeNormal, v1.EventRotatedInAzure, "Azure credentials is rotated")
	return &credentialsSet, keyIdsInUse, nil
}

func (a azureReconciler) validateCredentials(tx transaction) (bool, error) {
	if !tx.secrets.credentials.valid || tx.secrets.credentials.set == nil {
		return false, nil
	}

	logger.Debug("validating existing credentials for Azure application...")
	valid, err := a.AzureClient.ValidateCredentials(tx.toAzureTx(), *tx.secrets.credentials.set)
	if err != nil {
		return false, err
	}

	if valid {
		logger.Debug("existing credentials are valid and in sync with Azure")
	} else {
		logger.Warnf("existing credentials are not in sync with Azure")
	}

	return valid, nil
}

func (a azureReconciler) delete(tx transaction) error {
	logger.Info("deleting application in Azure AD...")
	exists, err := a.exists(tx)
	if err != nil {
		return err
	}
	if !exists {
		logger.Info("Azure application does not exist - skipping deletion")
		return nil
	}
	if err := a.AzureClient.Delete(tx.toAzureTx()); err != nil {
		return fmt.Errorf("failed to delete Azure application: %w", err)
	}
	logger.Info("Azure application successfully deleted")
	return nil
}

func (a azureReconciler) exists(tx transaction) (bool, error) {
	application, exists, err := a.AzureClient.Exists(tx.toAzureTx())
	if err != nil {
		return false, fmt.Errorf("looking up existence of azure application: %w", err)
	}

	if exists {
		tx.instance.Status.ClientId = *application.AppID
		tx.instance.Status.ObjectId = *application.ID

		sp, err := a.AzureClient.GetServicePrincipal(tx.toAzureTx())
		if err != nil {
			return false, fmt.Errorf("getting service principal for application: %w", err)
		}
		tx.instance.Status.ServicePrincipalId = *sp.ID

		tx.log.WithFields(log.Fields{
			"ClientID":           tx.instance.GetClientId(),
			"ObjectID":           tx.instance.GetObjectId(),
			"ServicePrincipalID": tx.instance.GetServicePrincipalId(),
		}).Debug("updated status fields with values from Azure")
	}

	return exists, nil
}

func (a azureReconciler) reportInvalid(tx transaction, preAuthApps azure.PreAuthorizedApps) {
	for _, app := range preAuthApps.Invalid {
		msg := fmt.Sprintf("Pre-authorized app '%s' was not found in the Azure AD tenant (%s), skipping assignment...", app.Name, a.Config.Azure.Tenant.String())
		tx.log.Warnf(msg)
		a.Recorder.Event(tx.instance, corev1.EventTypeWarning, v1.EventSkipped, msg)
	}
}
