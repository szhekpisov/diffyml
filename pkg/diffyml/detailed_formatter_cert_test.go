package diffyml

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

// makeTestCertPEM creates a PEM-encoded self-signed certificate for testing.
func makeTestCertPEM(t *testing.T, cn string, notBefore, notAfter time.Time, serial *big.Int) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}
	block := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}
	return string(pem.EncodeToMemory(block))
}

func TestDetailedFormatter_CertInspection_ModifiedCerts(t *testing.T) {
	fromCert := makeTestCertPEM(t, "old.example.com",
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(0xaabb))
	toCert := makeTestCertPEM(t, "new.example.com",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(0xccdd))

	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "tls.cert", Type: DiffModified, From: fromCert, To: toCert},
	}
	output := f.Format(diffs, opts)

	// Should show Certificate(...) summaries, not raw PEM
	if !strings.Contains(output, "Certificate(") {
		t.Errorf("expected Certificate() summary in output, got:\n%s", output)
	}
	if strings.Contains(output, "-----BEGIN CERTIFICATE-----") {
		t.Errorf("expected no raw PEM in output, got:\n%s", output)
	}
	if !strings.Contains(output, "old.example.com") {
		t.Errorf("expected old cert CN in output, got:\n%s", output)
	}
	if !strings.Contains(output, "new.example.com") {
		t.Errorf("expected new cert CN in output, got:\n%s", output)
	}
}

func TestDetailedFormatter_CertInspection_Disabled(t *testing.T) {
	fromCert := makeTestCertPEM(t, "old.example.com",
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(1))
	toCert := makeTestCertPEM(t, "new.example.com",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(2))

	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true
	opts.NoCertInspection = true

	diffs := []Difference{
		{Path: "tls.cert", Type: DiffModified, From: fromCert, To: toCert},
	}
	output := f.Format(diffs, opts)

	// Should show raw PEM, not Certificate() summary
	if strings.Contains(output, "Certificate(") {
		t.Errorf("expected no Certificate() summary when disabled, got:\n%s", output)
	}
	if !strings.Contains(output, "value change in multiline text") {
		t.Errorf("expected multiline text diff for raw PEM, got:\n%s", output)
	}
}

func TestDetailedFormatter_CertInspection_OnlyOneSideIsCert(t *testing.T) {
	certPEM := makeTestCertPEM(t, "example.com",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(1))

	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "tls.cert", Type: DiffModified, From: "not a cert", To: certPEM},
	}
	output := f.Format(diffs, opts)

	// When only one side is a cert, show as raw text (no cert inspection)
	if strings.Contains(output, "Certificate(") {
		t.Errorf("expected no Certificate() when only one side is cert, got:\n%s", output)
	}
}

func TestDetailedFormatter_CertInspection_AddedCert(t *testing.T) {
	certPEM := makeTestCertPEM(t, "added.example.com",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(0xab))

	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "secrets.tls-cert", Type: DiffAdded, To: certPEM},
	}
	output := f.Format(diffs, opts)

	if !strings.Contains(output, "Certificate(") {
		t.Errorf("expected Certificate() summary for added cert, got:\n%s", output)
	}
	if !strings.Contains(output, "added.example.com") {
		t.Errorf("expected cert CN in output, got:\n%s", output)
	}
}

func TestDetailedFormatter_CertInspection_RemovedCert(t *testing.T) {
	certPEM := makeTestCertPEM(t, "removed.example.com",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(0xcd))

	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	diffs := []Difference{
		{Path: "secrets.tls-cert", Type: DiffRemoved, From: certPEM},
	}
	output := f.Format(diffs, opts)

	if !strings.Contains(output, "Certificate(") {
		t.Errorf("expected Certificate() summary for removed cert, got:\n%s", output)
	}
	if !strings.Contains(output, "removed.example.com") {
		t.Errorf("expected cert CN in output, got:\n%s", output)
	}
}

func TestDetailedFormatter_CertInspection_AddedCert_Disabled(t *testing.T) {
	certPEM := makeTestCertPEM(t, "added.example.com",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(1))

	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true
	opts.NoCertInspection = true

	diffs := []Difference{
		{Path: "secrets.tls-cert", Type: DiffAdded, To: certPEM},
	}
	output := f.Format(diffs, opts)

	if strings.Contains(output, "Certificate(") {
		t.Errorf("expected no Certificate() when inspection disabled, got:\n%s", output)
	}
	if !strings.Contains(output, "-----BEGIN CERTIFICATE-----") {
		t.Errorf("expected raw PEM when inspection disabled, got:\n%s", output)
	}
}

func TestDetailedFormatter_CertInspection_NonStringValues(t *testing.T) {
	f, _ := FormatterByName("detailed")
	opts := DefaultFormatOptions()
	opts.OmitHeader = true

	// Non-string values should not trigger cert inspection
	diffs := []Difference{
		{Path: "config.port", Type: DiffModified, From: 8080, To: 9090},
	}
	output := f.Format(diffs, opts)

	if strings.Contains(output, "Certificate(") {
		t.Errorf("expected no cert inspection for non-string values, got:\n%s", output)
	}
	if !strings.Contains(output, "value change") {
		t.Errorf("expected normal value change output, got:\n%s", output)
	}
}
