package azureadapplication

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/annotations"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type namespaces struct {
	*Reconciler
}

var (
	namespaceCache = make(map[string]corev1.Namespace)
)

const (
	sharedNamespaceLabelKey = "shared"
)

func (r *Reconciler) namespaces() namespaces {
	return namespaces{r}
}

func (n namespaces) process(tx *transaction) (bool, error) {
	if tx.options.Namespace.HasIgnoreAnnotation {
		logger.Debug(fmt.Sprintf("Resource is annotated with '%s'. Skipping processing...", annotations.NotInTeamNamespaceKey))
		return true, nil
	}

	inSharedNamespace, err := n.inSharedNamespace(tx)
	if err != nil {
		return inSharedNamespace, err
	}

	if !inSharedNamespace {
		return false, nil
	}

	msg := fmt.Sprintf("ERROR: Expected resource in team namespace, but was found in namespace '%s'. Azure application and secrets will not be processed.", tx.instance.Namespace)
	logger.Error(msg)
	annotations.SetAnnotation(tx.instance, annotations.NotInTeamNamespaceKey, strconv.FormatBool(true))
	n.reportEvent(*tx, corev1.EventTypeWarning, v1.EventNotInTeamNamespace, msg)

	if err := n.Client.Status().Update(tx.ctx, tx.instance); err != nil {
		return inSharedNamespace, fmt.Errorf("failed to update resource with skip flag: %w", err)
	}

	if err := n.Client.Update(tx.ctx, tx.instance); err != nil {
		return inSharedNamespace, fmt.Errorf("failed to update resource with skip flag: %w", err)
	}

	return inSharedNamespace, nil
}

func (n namespaces) inSharedNamespace(tx *transaction) (bool, error) {
	namespaceName := tx.instance.GetNamespace()

	namespace, found := namespaceCache[namespaceName]

	var err error

	if !found {
		namespace, err = n.getNamespace(tx.ctx, namespaceName)
	}
	if err != nil {
		return false, fmt.Errorf("fetching namespace: %w", err)
	}

	return n.isSharedNamespace(namespace)
}

func (n namespaces) getNamespace(ctx context.Context, namespaceName string) (corev1.Namespace, error) {
	var namespace corev1.Namespace

	err := n.Reader.Get(ctx, client.ObjectKey{
		Name: namespaceName,
	}, &namespace)

	if err != nil {
		return namespace, err
	}

	return namespace, nil
}

func (n namespaces) isSharedNamespace(namespace corev1.Namespace) (bool, error) {
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
