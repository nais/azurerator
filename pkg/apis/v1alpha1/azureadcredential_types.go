package v1alpha1

// +groupName="nais.io"

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

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
	// Conditions lists the latest available observations of the object's current state
	Conditions []Condition `json:"conditions,omitempty"`
	// PasswordKeyId is the key ID for the latest valid password credential
	PasswordKeyId string `json:"passwordKeyId"`
	// CertificateKeyId is the certificate ID for the latest valid certificate credential
	CertificateKeyId string `json:"certificateKeyId"`
	// SynchronizationHash is the hash of the AzureAdCredential object
	SynchronizationHash string `json:"synchronizationHash,omitempty"`
	// SynchronizationTime is the time when the resource completed synchronization
	SynchronizationTime metav1.Time `json:"synchronizationTime,omitempty"`
}

type Condition struct {
	// Type is the type of condition for this resource
	// +kubebuilder:validation:Enum=Initializing;Completed;Failed
	Type ConditionType `json:"type"`
	// Status is the status of the condition, one of True, False, Unknown
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status ConditionStatus `json:"status"`
	// Reason is a one-word CamelCase reason for the condition's last transition
	Reason string `json:"reason,omitempty"`
	// Message is a human-readable message indicating details about last transition
	Message string `json:"message,omitempty"`
	// LastHeartbeatTIme is the last time we got an update on a given condition
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
	// LastTransitionTime is the last time the condition transit from one status to another
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

type ConditionType string

func (c Condition) Reconciled() bool {
	return c.Type == Completed && c.Status == True
}

const (
	Initializating ConditionType = "Initializing"
	Completed      ConditionType = "Completed"
	Failed         ConditionType = "Failed"
)

type ConditionStatus string

const (
	True    ConditionStatus = "True"
	False   ConditionStatus = "False"
	Unknown ConditionStatus = "Unknown"
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
