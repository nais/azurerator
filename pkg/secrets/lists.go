package secrets

import (
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/liberator/pkg/kubernetes"
)

func WithIdsFromUsedSecrets(a azure.ApplicationResult, s kubernetes.SecretLists) azure.ApplicationResult {
	passwordIds := make([]string, 0)
	certificateIds := make([]string, 0)
	for _, sec := range s.Used.Items {
		certificateId := string(sec.Data[CertificateIdKey])
		if len(certificateId) > 0 {
			certificateIds = append(certificateIds, certificateId)
		}
		passwordId := string(sec.Data[PasswordIdKey])
		if len(passwordId) > 0 {
			passwordIds = append(passwordIds, passwordId)
		}
	}
	a.Password.KeyId.AllInUse = passwordIds
	a.Certificate.KeyId.AllInUse = certificateIds
	return a
}
