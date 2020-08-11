package v1

// +groupName="nais.io"

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=azureapp
// +kubebuilder:subresource:status

// AzureAdApplication is the Schema for the AzureAdApplications API
type AzureAdApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureAdApplicationSpec   `json:"spec,omitempty"`
	Status AzureAdApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AzureAdApplicationList contains a list of AzureAdApplication
type AzureAdApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureAdApplication `json:"items"`
}

// AzureAdApplicationSpec defines the desired state of AzureAdApplication
type AzureAdApplicationSpec struct {
	ReplyUrls                 []AzureAdReplyUrl                 `json:"replyUrls,omitempty"`
	PreAuthorizedApplications []AzureAdPreAuthorizedApplication `json:"preAuthorizedApplications,omitempty"`
	// LogoutUrl is the URL where Azure AD sends a request to have the application clear the user's session data.
	// This is required if single sign-out should work correctly. Must start with 'https'
	LogoutUrl string `json:"logoutUrl,omitempty"`
	// SecretName is the name of the resulting Secret resource to be created
	SecretName string `json:"secretName"`
}

// AzureAdApplicationStatus defines the observed state of AzureAdApplication
type AzureAdApplicationStatus struct {
	// Synchronized denotes whether the provisioning of the AzureAdApplication has been successfully completed or not
	Synchronized bool `json:"synchronized"`
	// Timestamp is the last time the Status subresource was updated
	Timestamp metav1.Time `json:"timestamp,omitempty"`
	// ProvisionHash is the hash of the AzureAdApplication object
	ProvisionHash string `json:"provisionHash,omitempty"`
	// CorrelationId is the ID referencing the processing transaction last performed on this resource
	CorrelationId string `json:"correlationId"`
	// PasswordKeyIds is the list of key IDs for the latest valid password credentials in use
	PasswordKeyIds []string `json:"passwordKeyIds"`
	// CertificateKeyIds is the list of key IDs for the latest valid certificate credentials in use
	CertificateKeyIds []string `json:"certificateKeyIds"`
	// ClientId is the Azure application client ID
	ClientId string `json:"clientId"`
	// ObjectId is the Azure AD Application object ID
	ObjectId string `json:"objectId"`
	// ServicePrincipalId is the Azure applications service principal object ID
	ServicePrincipalId string `json:"servicePrincipalId"`
}

// AzureAdReplyUrl defines the valid reply URLs for callbacks after OIDC flows for this application
type AzureAdReplyUrl struct {
	Url string `json:"url,omitempty"`
}

// AzureAdPreAuthorizedApplication describes an application that are allowed to request an on-behalf-of token for this application
type AzureAdPreAuthorizedApplication struct {
	Application string `json:"application"`
	Namespace   string `json:"namespace"`
	Cluster     string `json:"cluster"`
}

const (
	LabelSkipKey   = "skip"
	LabelSkipValue = "true"
)

func init() {
	SchemeBuilder.Register(&AzureAdApplication{}, &AzureAdApplicationList{})
}
