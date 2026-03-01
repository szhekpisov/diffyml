package diffyml

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"
)

// generateTestCert creates a PEM-encoded self-signed certificate with the given parameters.
func generateTestCert(t *testing.T, cn string, org []string, dnsNames []string, notBefore, notAfter time.Time, serial *big.Int) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: org,
		},
		DNSNames:  dnsNames,
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}
	block := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}
	return string(pem.EncodeToMemory(block))
}

// --- IsPEMCertificate tests ---

func TestIsPEMCertificate_ValidCert(t *testing.T) {
	certPEM := generateTestCert(t, "example.com", nil, nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		big.NewInt(12345))
	if !IsPEMCertificate(certPEM) {
		t.Error("expected valid PEM certificate to be detected")
	}
}

func TestIsPEMCertificate_WhitespacePadded(t *testing.T) {
	certPEM := generateTestCert(t, "example.com", nil, nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		big.NewInt(1))
	padded := "\n  " + certPEM + "\n  "
	if !IsPEMCertificate(padded) {
		t.Error("expected whitespace-padded PEM certificate to be detected")
	}
}

func TestIsPEMCertificate_EmptyString(t *testing.T) {
	if IsPEMCertificate("") {
		t.Error("expected empty string to not be detected as certificate")
	}
}

func TestIsPEMCertificate_PlainText(t *testing.T) {
	if IsPEMCertificate("just a regular string value") {
		t.Error("expected plain text to not be detected as certificate")
	}
}

func TestIsPEMCertificate_InvalidPEM(t *testing.T) {
	invalid := "-----BEGIN CERTIFICATE-----\nnot-valid-base64\n-----END CERTIFICATE-----\n"
	if IsPEMCertificate(invalid) {
		t.Error("expected invalid PEM data to not be detected as certificate")
	}
}

func TestIsPEMCertificate_RSAKey(t *testing.T) {
	rsaKey := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF+PYPLp+Jy7OYG1pLkMbBLNh6sk\n-----END RSA PRIVATE KEY-----\n"
	if IsPEMCertificate(rsaKey) {
		t.Error("expected RSA key PEM block to not be detected as certificate")
	}
}

func TestIsPEMCertificate_CertificateChain(t *testing.T) {
	cert1 := generateTestCert(t, "leaf.example.com", nil, nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		big.NewInt(1))
	cert2 := generateTestCert(t, "ca.example.com", nil, nil,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		big.NewInt(2))
	chain := cert1 + cert2
	if IsPEMCertificate(chain) {
		t.Error("expected certificate chain (multiple PEM blocks) to not be detected")
	}
}

func TestIsPEMCertificate_EmbeddedInText(t *testing.T) {
	certPEM := generateTestCert(t, "example.com", nil, nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		big.NewInt(1))
	embedded := "Here is my certificate:\n" + certPEM + "\nPlease use it."
	if IsPEMCertificate(embedded) {
		t.Error("expected PEM embedded in larger text to not be detected")
	}
}

// --- FormatCertificate tests ---

func TestFormatCertificate_ValidCert(t *testing.T) {
	serial := big.NewInt(0x1234abcd)
	certPEM := generateTestCert(t, "example.com", nil, nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		serial)

	result := FormatCertificate(certPEM)

	expected := "Certificate(CN=example.com, Issuer=example.com, Valid=2026-01-01..2026-04-01, Serial=1234abcd)"
	if result != expected {
		t.Errorf("unexpected format:\n  got:  %s\n  want: %s", result, expected)
	}
}

func TestFormatCertificate_EmptyCN_FallbackToOrg(t *testing.T) {
	serial := big.NewInt(42)
	certPEM := generateTestCert(t, "", []string{"My Org"}, nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		serial)

	result := FormatCertificate(certPEM)

	if !strings.Contains(result, "CN=My Org") {
		t.Errorf("expected fallback to Organization, got: %s", result)
	}
}

func TestFormatCertificate_EmptyCN_FallbackToSAN(t *testing.T) {
	serial := big.NewInt(99)
	certPEM := generateTestCert(t, "", nil, []string{"san.example.com"},
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		serial)

	result := FormatCertificate(certPEM)

	if !strings.Contains(result, "CN=san.example.com") {
		t.Errorf("expected fallback to DNS SAN, got: %s", result)
	}
}

func TestFormatCertificate_InvalidCert_ReturnsOriginal(t *testing.T) {
	original := "not a certificate at all"
	result := FormatCertificate(original)
	if result != original {
		t.Errorf("expected original string returned for non-cert, got: %s", result)
	}
}

func TestFormatCertificate_CorruptedPEM_ReturnsOriginal(t *testing.T) {
	corrupted := "-----BEGIN CERTIFICATE-----\ngarbage-data!!!\n-----END CERTIFICATE-----\n"
	result := FormatCertificate(corrupted)
	if result != corrupted {
		t.Errorf("expected original string returned for corrupted PEM, got: %s", result)
	}
}

func TestFormatCertificate_SerialNumberHex(t *testing.T) {
	serial := big.NewInt(0xff)
	certPEM := generateTestCert(t, "test.com", nil, nil,
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC),
		serial)

	result := FormatCertificate(certPEM)

	if !strings.Contains(result, "Serial=ff") {
		t.Errorf("expected lowercase hex serial, got: %s", result)
	}
}

func TestFormatCertificate_IssuerCN(t *testing.T) {
	// Self-signed: issuer CN equals subject CN
	certPEM := generateTestCert(t, "myissuer.com", nil, nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(1))

	result := FormatCertificate(certPEM)

	if !strings.Contains(result, "Issuer=myissuer.com") {
		t.Errorf("expected issuer CN in output, got: %s", result)
	}
}

func TestFormatCertificate_OutputFormat(t *testing.T) {
	certPEM := generateTestCert(t, "test.com", nil, nil,
		time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		big.NewInt(1))

	result := FormatCertificate(certPEM)

	if !strings.HasPrefix(result, "Certificate(") {
		t.Errorf("expected output to start with 'Certificate(', got: %s", result)
	}
	if !strings.HasSuffix(result, ")") {
		t.Errorf("expected output to end with ')', got: %s", result)
	}
	if strings.Contains(result, "\n") {
		t.Errorf("expected single-line output, got: %s", result)
	}

	// Verify all expected fields are present
	for _, field := range []string{"CN=", "Issuer=", "Valid=", "Serial="} {
		if !strings.Contains(result, field) {
			t.Errorf("expected %q in output, got: %s", field, result)
		}
	}
}

func TestIsPEMCertificate_CorruptedDER(t *testing.T) {
	// Valid PEM structure but the DER content is not a valid certificate
	block := &pem.Block{Type: "CERTIFICATE", Bytes: []byte("not-a-real-der-cert")}
	badCert := string(pem.EncodeToMemory(block))

	if IsPEMCertificate(badCert) {
		t.Error("expected corrupted DER data to not be detected as certificate")
	}
}

func TestFormatCertificate_IssuerFallback(t *testing.T) {
	// When issuer CN is empty but issuer org is set, the issuer should use org
	// Since generateTestCert creates self-signed certs, we test with empty CN + org
	serial := big.NewInt(7)
	certPEM := generateTestCert(t, "", []string{"Issuer Org"}, nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		serial)

	result := FormatCertificate(certPEM)

	// Self-signed: issuer = subject, so Issuer should also fall back
	if !strings.Contains(result, fmt.Sprintf("Issuer=%s", "Issuer Org")) {
		t.Errorf("expected issuer fallback to Organization, got: %s", result)
	}
}
