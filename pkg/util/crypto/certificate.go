package crypto

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

func GenerateCertificate(template *x509.Certificate, keyPair KeyPair) (*x509.Certificate, error) {
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, keyPair.Public, keyPair.Private)
	if err != nil {
		return nil, fmt.Errorf("failed to generate the certificate for key: %w", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from DER data: %w", err)
	}
	return cert, nil
}

func CertificateTemplate(application v1.AzureAdApplication) *x509.Certificate {
	notBefore := time.Now()

	var notAfter time.Time
	if application.Spec.SecretProtected {
		notAfter = notBefore.AddDate(99, 0, 0)
	} else {
		notAfter = notBefore.AddDate(1, 0, 0)
	}

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
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
}

func ConvertToPem(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}
