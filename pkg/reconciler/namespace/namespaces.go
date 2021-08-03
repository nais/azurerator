package namespace

import (
	"context"
	"fmt"
	"strconv"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/reconciler"
	"github.com/nais/azureator/pkg/transaction"
)

type namespaceReconciler struct {
	reconciler.AzureAdApplication
	client client.Client
}

func NewNamespaceReconciler(reconciler reconciler.AzureAdApplication, client client.Client) reconciler.Namespace {
	return namespaceReconciler{
		AzureAdApplication: reconciler,
		client:             client,
	}
}

var (
	namespaceCache = make(map[string]corev1.Namespace)
)

const (
	sharedNamespaceLabelKey = "shared"
)

func (n namespaceReconciler) Process(tx *transaction.Transaction) (bool, error) {
	if tx.Options.Namespace.HasIgnoreAnnotation {
		tx.Logger.Debug(fmt.Sprintf("Resource is annotated with '%s'. Skipping processing...", annotations.NotInTeamNamespaceKey))
		return true, nil
	}

	inSharedNamespace, err := n.inSharedNamespace(tx)
	if err != nil {
		return inSharedNamespace, err
	}

	if !inSharedNamespace {
		return false, nil
	}

	msg := fmt.Sprintf("ERROR: Expected resource in team namespace, but was found in namespace '%s'. Azure application and secrets will not be processed.", tx.Instance.Namespace)
	tx.Logger.Error(msg)
	annotations.SetAnnotation(tx.Instance, annotations.NotInTeamNamespaceKey, strconv.FormatBool(true))
	n.ReportEvent(*tx, corev1.EventTypeWarning, v1.EventNotInTeamNamespace, msg)

	if err := n.client.Status().Update(tx.Ctx, tx.Instance); err != nil {
		return inSharedNamespace, fmt.Errorf("failed to update resource with skip flag: %w", err)
	}

	if err := n.client.Update(tx.Ctx, tx.Instance); err != nil {
		return inSharedNamespace, fmt.Errorf("failed to update resource with skip flag: %w", err)
	}

	return inSharedNamespace, nil
}

func (n namespaceReconciler) inSharedNamespace(tx *transaction.Transaction) (bool, error) {
	namespaceName := tx.Instance.GetNamespace()

	namespace, found := namespaceCache[namespaceName]

	var err error

	if !found {
		namespace, err = n.getNamespace(tx.Ctx, namespaceName)
		namespaceCache[namespaceName] = namespace
	}
	if err != nil {
		return false, fmt.Errorf("fetching namespace: %w", err)
	}

	return n.isSharedNamespace(namespace)
}

func (n namespaceReconciler) getNamespace(ctx context.Context, namespaceName string) (corev1.Namespace, error) {
	var namespace corev1.Namespace

	err := n.client.Get(ctx, client.ObjectKey{
		Name: namespaceName,
	}, &namespace)

	if err != nil {
		return namespace, err
	}

	return namespace, nil
}

func (n namespaceReconciler) isSharedNamespace(namespace corev1.Namespace) (bool, error) {
	stringValue, found := namespace.GetLabels()[sharedNamespaceLabelKey]
	if !found {
		return false, nil
	}

	shared, err := strconv.ParseBool(stringValue)
	if err != nil {
		return false, err
	}

	return shared, nil
}
