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
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ReplyUrls                 []AzureAdReplyUrl                 `json:"replyUrls,omitempty"`
	PreAuthorizedApplications []AzureAdPreAuthorizedApplication `json:"preAuthorizedApplications,omitempty"`
}

// AzureAdCredentialStatus defines the observed state of AzureAdCredential
type AzureAdCredentialStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

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
