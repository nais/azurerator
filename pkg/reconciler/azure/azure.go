package azure

import (
	"fmt"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/nais/msgraph.go/ptr"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/event"
	"github.com/nais/azureator/pkg/kafka"
	"github.com/nais/azureator/pkg/metrics"
	"github.com/nais/azureator/pkg/reconciler"
	"github.com/nais/azureator/pkg/transaction"
)

type azureReconciler struct {
	reconciler.AzureAdApplication
	azureClient azure.Client
	config      config.Config
	kafkaClient kafka.Client
	recorder    record.EventRecorder
}

func NewAzureReconciler(
	reconciler reconciler.AzureAdApplication,
	azureClient azure.Client,
	config config.Config,
	recorder record.EventRecorder,
	kafkaClient kafka.Client,
) reconciler.Azure {
	return azureReconciler{
		AzureAdApplication: reconciler,
		azureClient:        azureClient,
		config:             config,
		kafkaClient:        kafkaClient,
		recorder:           recorder,
	}
}

func (a azureReconciler) Process(tx transaction.Transaction) (*result.Application, error) {
	var applicationResult *result.Application

	exists, err := a.Exists(tx)
	if err != nil {
		return nil, fmt.Errorf("looking up existence of application: %w", err)
	}

	if !exists {
		applicationResult, err = a.create(tx)
	} else if tx.Options.Process.Azure.Synchronize {
		applicationResult, err = a.update(tx)
	} else {
		applicationResult, err = a.notModified(tx)
	}
	if err != nil {
		return nil, err
	}

	if applicationResult.IsModified() {
		a.reportPreAuthorizedApplicationStatus(tx, applicationResult.PreAuthorizedApps)
	}

	err = a.produceEvent(tx, applicationResult)
	if err != nil {
		tx.Logger.Errorf("producing kafka event: %+v", err)
	}

	return applicationResult, nil
}

func (a azureReconciler) create(tx transaction.Transaction) (*result.Application, error) {
	tx.Logger.Info("Azure application not found, registering...")

	applicationResult, err := a.azureClient.Create(tx.ToAzureTx())
	if err != nil {
		return nil, fmt.Errorf("creating azure application: %w", err)
	}

	metrics.IncWithNamespaceLabel(metrics.AzureAppsCreatedCount, tx.Instance.Namespace)
	a.ReportEvent(tx, corev1.EventTypeNormal, v1.EventCreatedInAzure, "Azure application is created")

	tx.Instance.Status.ClientId = applicationResult.ClientId
	tx.Instance.Status.ObjectId = applicationResult.ObjectId
	tx.Instance.Status.ServicePrincipalId = applicationResult.ServicePrincipalId

	return applicationResult, nil
}

func (a azureReconciler) update(tx transaction.Transaction) (*result.Application, error) {
	tx.Logger.Info("Azure application already exists, updating...")

	applicationResult, err := a.azureClient.Update(tx.ToAzureTx())
	if err != nil {
		return nil, fmt.Errorf("updating azure application: %w", err)
	}

	metrics.IncWithNamespaceLabel(metrics.AzureAppsUpdatedCount, tx.Instance.Namespace)
	a.ReportEvent(tx, corev1.EventTypeNormal, v1.EventUpdatedInAzure, "Azure application is updated")

	return applicationResult, nil
}

func (a azureReconciler) notModified(tx transaction.Transaction) (*result.Application, error) {
	apps, err := a.azureClient.GetPreAuthorizedApps(tx.ToAzureTx())
	if err != nil {
		return nil, fmt.Errorf("fetching pre-authorized apps: %w", err)
	}
	return &result.Application{
		ClientId:           tx.Instance.Status.ClientId,
		ObjectId:           tx.Instance.Status.ObjectId,
		ServicePrincipalId: tx.Instance.Status.ServicePrincipalId,
		PreAuthorizedApps:  *apps,
		Tenant:             a.config.Azure.Tenant.Id,
		Result:             result.OperationNotModified,
	}, nil
}

func (a azureReconciler) produceEvent(tx transaction.Transaction, result *result.Application) error {
	if !a.config.Kafka.Enabled {
		return nil
	}

	if result.IsCreated() {
		e := event.NewEvent(tx.ID, event.Created, tx.Instance)
		return a.kafkaClient.ProduceEvent(tx.Ctx, e)
	}

	return nil
}

func (a azureReconciler) AddCredentials(tx transaction.Transaction, keyIdsInUse credentials.KeyIdsInUse) (*credentials.Set, credentials.KeyIdsInUse, error) {
	tx.Logger.Info("adding credentials for Azure application...")

	credentialsSet, err := a.azureClient.AddCredentials(tx.ToAzureTx())
	if err != nil {
		return nil, credentials.KeyIdsInUse{}, err
	}

	tx.Logger.Info("successfully added credentials for Azure application")

	keyIdsInUse.Certificate = append(keyIdsInUse.Certificate, credentialsSet.Current.Certificate.KeyId)
	keyIdsInUse.Password = append(keyIdsInUse.Password, credentialsSet.Current.Password.KeyId)
	return &credentialsSet, keyIdsInUse, nil
}

func (a azureReconciler) RotateCredentials(tx transaction.Transaction, existing credentials.Set, keyIdsInUse credentials.KeyIdsInUse) (*credentials.Set, credentials.KeyIdsInUse, error) {
	tx.Logger.Info("rotating credentials for Azure application...")

	credentialsSet, err := a.azureClient.RotateCredentials(tx.ToAzureTx(), existing, keyIdsInUse)
	if err != nil {
		return nil, credentials.KeyIdsInUse{}, err
	}

	tx.Logger.Info("successfully rotated credentials for Azure application")

	keyIdsInUse.Certificate = append(keyIdsInUse.Certificate, credentialsSet.Current.Certificate.KeyId)
	keyIdsInUse.Password = append(keyIdsInUse.Password, credentialsSet.Current.Password.KeyId)

	metrics.IncWithNamespaceLabel(metrics.AzureAppsRotatedCount, tx.Instance.Namespace)
	a.ReportEvent(tx, corev1.EventTypeNormal, v1.EventRotatedInAzure, "Azure credentials is rotated")
	return &credentialsSet, keyIdsInUse, nil
}

func (a azureReconciler) PurgeCredentials(tx transaction.Transaction) error {
	exists, err := a.Exists(tx)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	tx.Logger.Debug("purging existing credentials for Azure application...")
	return a.azureClient.PurgeCredentials(tx.ToAzureTx())
}

func (a azureReconciler) ValidateCredentials(tx transaction.Transaction) (bool, error) {
	exists, err := a.Exists(tx)
	if err != nil {
		return false, err
	}

	if !exists || !tx.Options.Process.Secret.Valid {
		return false, nil
	}

	tx.Logger.Debug("validating existing credentials for Azure application...")
	valid, err := a.azureClient.ValidateCredentials(tx.ToAzureTx(), *tx.Secrets.Credentials.Set)
	if err != nil {
		return false, err
	}

	if valid {
		tx.Logger.Debug("existing credentials are valid and in sync with Azure")
	} else {
		tx.Logger.Warnf("existing credentials are not in sync with Azure")
	}

	return valid, nil
}

func (a azureReconciler) Delete(tx transaction.Transaction) error {
	tx.Logger.Info("deleting application in Azure AD...")
	exists, err := a.Exists(tx)
	if err != nil {
		return err
	}
	if !exists {
		tx.Logger.Info("Azure application does not exist - skipping deletion")
		return nil
	}
	if err := a.azureClient.Delete(tx.ToAzureTx()); err != nil {
		return fmt.Errorf("failed to delete Azure application: %w", err)
	}
	tx.Logger.Info("Azure application successfully deleted")
	return nil
}

func (a azureReconciler) Exists(tx transaction.Transaction) (bool, error) {
	application, exists, err := a.azureClient.Exists(tx.ToAzureTx())
	if err != nil {
		return false, fmt.Errorf("looking up existence of azure application: %w", err)
	}

	if exists {
		tx.Instance.Status.ClientId = *application.AppID
		tx.Instance.Status.ObjectId = *application.ID

		sp, err := a.azureClient.GetServicePrincipal(tx.ToAzureTx())
		if err != nil {
			return false, fmt.Errorf("getting service principal for application: %w", err)
		}
		tx.Instance.Status.ServicePrincipalId = *sp.ID

		tx.Logger.WithFields(log.Fields{
			"ClientID":           tx.Instance.GetClientId(),
			"ObjectID":           tx.Instance.GetObjectId(),
			"ServicePrincipalID": tx.Instance.GetServicePrincipalId(),
		}).Debug("updated status fields with values from Azure")
	}

	return exists, nil
}

func (a azureReconciler) ProcessOrphaned(tx transaction.Transaction) error {
	exists, err := a.Exists(tx)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	tx.Logger.Warnf("orphaned resource '%s' found in tenant %s", kubernetes.UniformResourceName(tx.Instance), a.config.Azure.Tenant)
	metrics.IncWithNamespaceLabel(metrics.AzureAppOrphanedTotal, tx.Instance.GetNamespace())

	if tx.Options.Process.Azure.CleanupOrphans {
		err = a.Delete(tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a azureReconciler) reportPreAuthorizedApplicationStatus(tx transaction.Transaction, preAuthApps result.PreAuthorizedApps) {
	unassigned := make([]v1.AzureAdPreAuthorizedApp, 0)
	assigned := make([]v1.AzureAdPreAuthorizedApp, 0)

	for _, app := range preAuthApps.Valid {
		rule := app.AccessPolicyRule
		assigned = append(assigned, v1.AzureAdPreAuthorizedApp{
			AccessPolicyRule:         &rule,
			ClientID:                 app.ClientId,
			ServicePrincipalObjectID: app.ObjectId,
		})
	}

	for _, app := range preAuthApps.Invalid {
		message := fmt.Sprintf(
			"WARNING: Application '%s' was not found in the Azure AD tenant (%s) and will _NOT_ be pre-authorized.",
			app.Name, a.config.Azure.Tenant.String(),
		)

		tx.Logger.Warnf(message)
		a.recorder.Eventf(tx.Instance, corev1.EventTypeNormal, v1.EventSkipped, message)

		rule := app.AccessPolicyRule
		unassigned = append(unassigned, v1.AzureAdPreAuthorizedApp{
			AccessPolicyRule:         &rule,
			ClientID:                 app.ClientId,
			ServicePrincipalObjectID: app.ObjectId,
			Reason:                   message,
		})
	}

	tx.Instance.Status.PreAuthorizedApps = &v1.AzureAdPreAuthorizedAppsStatus{
		Assigned:        assigned,
		AssignedCount:   ptr.Int(len(assigned)),
		Unassigned:      unassigned,
		UnassignedCount: ptr.Int(len(unassigned)),
	}
}
