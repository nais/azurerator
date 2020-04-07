package v1alpha1

// +groupName="nais.io"

import (
	"encoding/json"
	"fmt"

	hash "github.com/mitchellh/hashstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=aad
// +kubebuilder:subresource:status

// AzureAdCredential is the Schema for the azureadcredentials API
type AzureAdCredential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureAdCredentialSpec   `json:"spec,omitempty"`
	Status AzureAdCredentialStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AzureAdCredentialList contains a list of AzureAdCredential
type AzureAdCredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureAdCredential `json:"items"`
}

// AzureAdCredentialSpec defines the desired state of AzureAdCredential
type AzureAdCredentialSpec struct {
	ReplyUrls                 []AzureAdReplyUrl                 `json:"replyUrls,omitempty"`
	PreAuthorizedApplications []AzureAdPreAuthorizedApplication `json:"preAuthorizedApplications,omitempty"`
}

// AzureAdCredentialStatus defines the observed state of AzureAdCredential
type AzureAdCredentialStatus struct {
	// ProvisionStatus denotes whether the provisioning of the AzureAdCredential has been initialized, completed successfully, or if the status is unknown
	// +kubebuilder:validation:Enum=initializing;unknown;complete
	ProvisionStatus ProvisionStatus `json:"provisionStatus"`
	// ProvisionState is a one-word CamelCase machine-readable representation of the current state of the object
	ProvisionState ProvisionState `json:"provisionState"`
	// LastStateTransitionTime is the last time the state transitioned from one state to another
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// ProvisionHash is the hash of the AzureAdCredential object
	ProvisionHash string `json:"provisionHash,omitempty"`
	// ProvisionTime is the time when the resource completed synchronization
	ProvisionTime metav1.Time `json:"provisionTime,omitempty"`
	// PasswordKeyId is the key ID for the latest valid password credential
	PasswordKeyId string `json:"passwordKeyId"`
	// CertificateKeyId is the certificate ID for the latest valid certificate credential
	CertificateKeyId string `json:"certificateKeyId"`
	// ClientId is the Azure application client ID
	ClientId string `json:"clientId"`
	// ApplicationObjectId is the Azure object ID
	ObjectId string `json:"objectId"`
}

type ProvisionState string

const (
	StateNewProvisioning    ProvisionState = "NewProvisioning"
	StateRotateProvisioning ProvisionState = "RotateProvisioning"
	StateRetrying           ProvisionState = "Retrying"
	StateProvisioned        ProvisionState = "Provisioned"
)

type ProvisionStatus string

const (
	Initializing ProvisionStatus = "initializing"
	Unknown      ProvisionStatus = "unknown"
	Complete     ProvisionStatus = "complete"
)

// AzureAdReplyUrl defines the valid reply URLs for callbacks after OIDC flows for this application
type AzureAdReplyUrl struct {
	Url string `json:"url,omitempty"`
}

// AzureAdPreAuthorizedApplication describes an application that are allowed to request an on-behalf-of token for this application
type AzureAdPreAuthorizedApplication struct {
	Name     string `json:"name,omitempty"`
	ClientId string `json:"clientId,omitempty"`
}

func init() {
	SchemeBuilder.Register(&AzureAdCredential{}, &AzureAdCredentialList{})
}

func (in *AzureAdCredential) StatusNewProvisioning() {
	in.Status.ProvisionStatus = Initializing
	in.Status.ProvisionState = StateNewProvisioning
	in.Status.LastTransitionTime = metav1.Now()
}

func (in *AzureAdCredential) StatusRotateProvisioning() {
	in.Status.ProvisionStatus = Initializing
	in.Status.ProvisionState = StateRotateProvisioning
	in.Status.LastTransitionTime = metav1.Now()
}

func (in *AzureAdCredential) StatusRetrying() {
	in.Status.ProvisionState = StateRetrying
	in.Status.LastTransitionTime = metav1.Now()
}

func (in *AzureAdCredential) StatusProvisioned() {
	in.Status.ProvisionStatus = Complete
	in.Status.ProvisionState = StateProvisioned
	in.Status.LastTransitionTime = metav1.Now()
	in.Status.ProvisionTime = metav1.Now()
}

func (in *AzureAdCredential) SetCertificateKeyId(keyId string) {
	in.Status.CertificateKeyId = keyId
}
func (in *AzureAdCredential) SetPasswordKeyId(keyId string) {
	in.Status.PasswordKeyId = keyId
}
func (in *AzureAdCredential) SetClientId(id string) {
	in.Status.ClientId = id
}
func (in *AzureAdCredential) SetObjectId(id string) {
	in.Status.ObjectId = id
}

func (in AzureAdCredential) Hash() (string, error) {
	// struct including the relevant fields for
	// creating a hash of an AzureAdCredential object
	var changeCause string
	if in.Annotations != nil {
		changeCause = in.Annotations["kubernetes.io/change-cause"]
	}
	relevantValues := struct {
		AzureAdCredentialSpec AzureAdCredentialSpec
		CertificateKeyId      string
		SecretKeyid           string
		ClientId              string
		ObjectId              string
		ChangeCause           string
	}{
		in.Spec,
		in.Status.CertificateKeyId,
		in.Status.PasswordKeyId,
		in.Status.ClientId,
		in.Status.ObjectId,
		changeCause,
	}

	marshalled, err := json.Marshal(relevantValues)
	if err != nil {
		return "", err
	}
	h, err := hash.Hash(marshalled, nil)
	return fmt.Sprintf("%x", h), err
}
