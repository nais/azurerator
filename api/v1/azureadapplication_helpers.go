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

func (in *AzureAdApplication) SetNotSynchronized() {
	in.Status.Synchronized = false
	in.Status.Timestamp = metav1.Now()
	if in.Status.PasswordKeyIds == nil {
		in.Status.PasswordKeyIds = make([]string, 0)
	}
	if in.Status.CertificateKeyIds == nil {
		in.Status.CertificateKeyIds = make([]string, 0)
	}
}

func (in *AzureAdApplication) SetSynchronized() {
	in.Status.Synchronized = true
	in.Status.Timestamp = metav1.Now()
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
	if hashUnchanged && in.Status.Synchronized {
		return true, nil
	}
	return false, nil
}

func (in *AzureAdApplication) UpdateHash() error {
	newHash, err := in.Hash()
	if err != nil {
		return fmt.Errorf("failed to calculate application hash: %w", err)
	}
	in.Status.ProvisionHash = newHash
	return nil
}

func (in *AzureAdApplication) HashUnchanged() (bool, error) {
	newHash, err := in.Hash()
	if err != nil {
		return false, fmt.Errorf("failed to calculate application hash: %w", err)
	}
	return in.Status.ProvisionHash == newHash, nil
}

func (in *AzureAdApplication) SetSkipAnnotation() {
	if in.ObjectMeta.Annotations == nil {
		in.ObjectMeta.Annotations = map[string]string{annotations.SkipKey: annotations.SkipValue}
	} else {
		in.ObjectMeta.Annotations[annotations.SkipKey] = annotations.SkipValue
	}
}

func (in AzureAdApplication) Hash() (string, error) {
	// struct including the relevant fields for
	// creating a hash of an AzureAdApplication object
	relevantValues := struct {
		AzureAdApplicationSpec AzureAdApplicationSpec
		CertificateKeyIds      []string
		SecretKeyIds           []string
		ClientId               string
		ObjectId               string
		ServicePrincipalId     string
	}{
		in.Spec,
		in.Status.CertificateKeyIds,
		in.Status.PasswordKeyIds,
		in.Status.ClientId,
		in.Status.ObjectId,
		in.Status.ServicePrincipalId,
	}

	marshalled, err := json.Marshal(relevantValues)
	if err != nil {
		return "", err
	}
	h, err := hash.Hash(marshalled, nil)
	return fmt.Sprintf("%x", h), err
}

func (in AzureAdApplication) GetUniqueName() string {
	return fmt.Sprintf("%s:%s:%s", in.ClusterName, in.Namespace, in.Name)
}
