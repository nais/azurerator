package v1

import (
	"encoding/json"
	"fmt"
	hash "github.com/mitchellh/hashstructure"
	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (in AzureAdPreAuthorizedApplication) GetUniqueName() string {
	return fmt.Sprintf("%s:%s:%s", in.Cluster, in.Namespace, in.Application)
}

func (in *AzureAdApplication) SetSynchronized() {
	in.Status.SynchronizationState = EventSynchronized
	now := metav1.Now()
	in.Status.SynchronizationTime = &now
}

func (in *AzureAdApplication) IsBeingDeleted() bool {
	return !in.ObjectMeta.DeletionTimestamp.IsZero()
}

func (in *AzureAdApplication) HasFinalizer(finalizerName string) bool {
	return util.ContainsString(in.ObjectMeta.Finalizers, finalizerName)
}

func (in *AzureAdApplication) AddFinalizer(finalizerName string) {
	in.ObjectMeta.Finalizers = append(in.ObjectMeta.Finalizers, finalizerName)
}

func (in *AzureAdApplication) RemoveFinalizer(finalizerName string) {
	in.ObjectMeta.Finalizers = util.RemoveString(in.ObjectMeta.Finalizers, finalizerName)
}

func (in *AzureAdApplication) IsUpToDate() (bool, error) {
	hashUnchanged, err := in.HashUnchanged()
	if err != nil {
		return false, err
	}
	return hashUnchanged, nil
}

func (in *AzureAdApplication) UpdateHash() error {
	newHash, err := in.Hash()
	if err != nil {
		return fmt.Errorf("failed to calculate application hash: %w", err)
	}
	in.Status.SynchronizationHash = newHash
	return nil
}

func (in *AzureAdApplication) HashUnchanged() (bool, error) {
	newHash, err := in.Hash()
	if err != nil {
		return false, fmt.Errorf("failed to calculate application hash: %w", err)
	}
	return in.Status.SynchronizationHash == newHash, nil
}

func (in *AzureAdApplication) SetSkipAnnotation() {
	annotations.SetAnnotation(in, annotations.SkipKey, annotations.SkipValue)
}

func (in AzureAdApplication) Hash() (string, error) {
	marshalled, err := json.Marshal(in.Spec)
	if err != nil {
		return "", err
	}
	h, err := hash.Hash(marshalled, nil)
	return fmt.Sprintf("%x", h), err
}

func (in AzureAdApplication) GetUniqueName() string {
	return fmt.Sprintf("%s:%s:%s", in.ClusterName, in.Namespace, in.Name)
}
