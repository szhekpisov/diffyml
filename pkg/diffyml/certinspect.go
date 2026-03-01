package diffyml

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
)

// IsPEMCertificate checks if a string is a PEM-encoded X.509 certificate.
// Returns true only if the trimmed string starts with the PEM CERTIFICATE header,
// decodes as a valid PEM block with type "CERTIFICATE", parses as a valid X.509
// certificate, and contains no additional PEM blocks (certificate chains return false).
func IsPEMCertificate(s string) bool {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "-----BEGIN CERTIFICATE-----") {
		return false
	}
	block, rest := pem.Decode([]byte(trimmed))
	if block == nil || block.Type != "CERTIFICATE" {
		return false
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return false
	}
	if strings.Contains(string(rest), "-----BEGIN CERTIFICATE-----") {
		return false
	}
	return true
}

// FormatCertificate parses a PEM-encoded certificate string and returns a
// human-readable single-line summary. Returns the original string unchanged
// if parsing fails.
func FormatCertificate(s string) string {
	block, _ := pem.Decode([]byte(strings.TrimSpace(s)))
	if block == nil {
		return s
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return s
	}

	cn := certName(cert.Subject.CommonName, cert.Subject.Organization, cert.DNSNames)
	issuer := certName(cert.Issuer.CommonName, cert.Issuer.Organization, nil)

	return fmt.Sprintf("Certificate(CN=%s, Issuer=%s, Valid=%s..%s, Serial=%s)",
		cn,
		issuer,
		cert.NotBefore.UTC().Format("2006-01-02"),
		cert.NotAfter.UTC().Format("2006-01-02"),
		cert.SerialNumber.Text(16),
	)
}

// certName returns the common name, falling back to the first organization
// or first DNS name if the common name is empty.
func certName(commonName string, orgs []string, dnsNames []string) string {
	if commonName != "" {
		return commonName
	}
	if len(orgs) > 0 {
		return orgs[0]
	}
	if len(dnsNames) > 0 {
		return dnsNames[0]
	}
	return ""
}
