package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

func GenerateCertificate(template *x509.Certificate, key *rsa.PrivateKey) (*x509.Certificate, error) {
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate the certificate for key: %w", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from DER data: %w", err)
	}
	return cert, nil
}

func Template(application v1alpha1.AzureAdCredential) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:            []string{"NO"},
			Province:           []string{"Oslo"},
			Locality:           []string{"Oslo"},
			Organization:       []string{"NAV (Arbeids- og velferdsdirektoratet"},
			OrganizationalUnit: []string{"NAV IT"},
			CommonName:         fmt.Sprintf("%s.%s.%s.azurerator.nais.io", application.Name, application.Namespace, application.ClusterName),
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
}

func ConvertToPem(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}