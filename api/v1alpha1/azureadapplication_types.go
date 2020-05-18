package v1alpha1

// +groupName="nais.io"

import (
	"encoding/json"
	"fmt"

	hash "github.com/mitchellh/hashstructure"
	"github.com/nais/azureator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=azuread
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
	// ConfigMapName is the name of the resulting ConfigMap resource to be created
	ConfigMapName string `json:"configMapName"`
}

// AzureAdApplicationStatus defines the observed state of AzureAdApplication
type AzureAdApplicationStatus struct {
	// UpToDate denotes whether the provisioning of the AzureAdApplication has been successfully completed or not
	UpToDate bool `json:"upToDate"`
	// ProvisionState is a one-word CamelCase machine-readable representation of the current state of the object
	// +kubebuilder:validation:Enum=New;Rotate;Retrying;Provisioned
	ProvisionState ProvisionState `json:"provisionState"`
	// ProvisionStateTime is the last time the state transitioned from one state to another
	ProvisionStateTime metav1.Time `json:"provisionStateTime,omitempty"`
	// ProvisionHash is the hash of the AzureAdApplication object
	ProvisionHash string `json:"provisionHash,omitempty"`
	// CorrelationId is the ID referencing the processing transaction last performed on this resource
	CorrelationId string `json:"correlationId"`
	// PasswordKeyId is the key ID for the latest valid password credential
	PasswordKeyId string `json:"passwordKeyId"`
	// CertificateKeyId is the certificate ID for the latest valid certificate credential
	CertificateKeyId string `json:"certificateKeyId"`
	// ClientId is the Azure application client ID
	ClientId string `json:"clientId"`
	// ObjectId is the Azure AD Application object ID
	ObjectId string `json:"objectId"`
	// ServicePrincipalId is the Azure applications service principal object ID
	ServicePrincipalId string `json:"servicePrincipalId"`
}

type ProvisionState string

const (
	New         ProvisionState = "New"
	Rotate      ProvisionState = "Rotate"
	Retrying    ProvisionState = "Retrying"
	Provisioned ProvisionState = "Provisioned"
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
	SchemeBuilder.Register(&AzureAdApplication{}, &AzureAdApplicationList{})
}

func (in *AzureAdApplication) SetStatusNew() {
	in.Status.UpToDate = false
	in.Status.ProvisionState = New
	in.Status.ProvisionStateTime = metav1.Now()
}

func (in *AzureAdApplication) SetStatusRotate() {
	in.Status.UpToDate = false
	in.Status.ProvisionState = Rotate
	in.Status.ProvisionStateTime = metav1.Now()
}

func (in *AzureAdApplication) SetStatusRetrying() {
	in.Status.UpToDate = false
	in.Status.ProvisionState = Retrying
	in.Status.ProvisionStateTime = metav1.Now()
}

func (in *AzureAdApplication) SetStatusProvisioned() {
	in.Status.UpToDate = true
	in.Status.ProvisionState = Provisioned
	in.Status.ProvisionStateTime = metav1.Now()
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
	if hashUnchanged && in.Status.UpToDate {
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

func (in AzureAdApplication) Hash() (string, error) {
	// struct including the relevant fields for
	// creating a hash of an AzureAdApplication object
	relevantValues := struct {
		AzureAdApplicationSpec AzureAdApplicationSpec
		CertificateKeyId       string
		SecretKeyid            string
		ClientId               string
		ObjectId               string
		ServicePrincipalId     string
	}{
		in.Spec,
		in.Status.CertificateKeyId,
		in.Status.PasswordKeyId,
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
