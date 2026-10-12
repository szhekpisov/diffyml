package diffyml

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"
)

// githubTestCerts returns two distinct PEM certificates, the shape of a
// rotation: same key path, different subject and validity window.
func githubTestCerts(t *testing.T) (from, to string) {
	t.Helper()
	from = makeTestCertPEM(t, "old.example.com",
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(0xaabb))
	to = makeTestCertPEM(t, "new.example.com",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		big.NewInt(0xccdd))
	return from, to
}

func TestGitHubFormatter_CertInspection_Modified(t *testing.T) {
	// A rotation used to render as a 30-line base64 line diff, because the
	// GitHub path reached the multiline check before considering certificates.
	fromCert, toCert := githubTestCerts(t)

	f := &GitHubFormatter{}
	msg := githubMessage(t, f.Format([]Difference{
		{Path: DiffPath{"data", "tls.crt"}, Type: DiffModified, From: fromCert, To: toCert},
	}, DefaultFormatOptions()))

	if strings.Contains(msg, "BEGIN CERTIFICATE") {
		t.Errorf("expected no raw PEM in the annotation, got: %s", msg)
	}
	if strings.Contains(msg, "changed in multiline text") {
		t.Errorf("expected the summary path, not the line diff, got: %s", msg)
	}
	for _, want := range []string{"changed from Certificate(", "old.example.com", "new.example.com"} {
		if !strings.Contains(msg, want) {
			t.Errorf("expected %q in the annotation, got: %s", want, msg)
		}
	}
}

func TestGitHubFormatter_CertInspection_SingleValues(t *testing.T) {
	cert, _ := githubTestCerts(t)

	tests := []struct {
		name string
		diff Difference
	}{
		{"added", Difference{Path: DiffPath{"data", "tls.crt"}, Type: DiffAdded, To: cert}},
		{"removed", Difference{Path: DiffPath{"data", "tls.crt"}, Type: DiffRemoved, From: cert}},
		{"unchanged", Difference{Path: DiffPath{"data", "tls.crt"}, Type: DiffUnchanged, To: cert}},
	}

	f := &GitHubFormatter{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := githubMessage(t, f.Format([]Difference{tt.diff}, DefaultFormatOptions()))

			if !strings.Contains(msg, "Certificate(") || !strings.Contains(msg, "old.example.com") {
				t.Errorf("expected a certificate summary, got: %s", msg)
			}
			if strings.Contains(msg, "BEGIN CERTIFICATE") {
				t.Errorf("expected no raw PEM in the annotation, got: %s", msg)
			}
		})
	}
}

func TestGitHubFormatter_CertInspection_Disabled(t *testing.T) {
	fromCert, toCert := githubTestCerts(t)

	opts := DefaultFormatOptions()
	opts.NoCertInspection = true

	// With inspection off a PEM is just a long multiline string, so the value
	// caps reach it like any other. One test certificate is well under
	// gitHubMaxValueLines; a bundle of three is over it.
	bundle := strings.Repeat(fromCert, 3)

	f := &GitHubFormatter{}
	diffs := []Difference{
		{Path: DiffPath{"data", "tls.crt"}, Type: DiffModified, From: fromCert, To: toCert},
		{Path: DiffPath{"data", "ca.crt"}, Type: DiffAdded, To: fromCert},
		{Path: DiffPath{"data", "bundle.crt"}, Type: DiffAdded, To: bundle},
	}
	output := f.Format(diffs, opts)

	if strings.Contains(output, "Certificate(") {
		t.Errorf("expected no summary when inspection is off, got: %s", output)
	}
	if !strings.Contains(output, escapeGitHubData("BEGIN CERTIFICATE")) {
		t.Errorf("expected the raw PEM when inspection is off, got: %s", output)
	}

	// One command per difference: the PEM's line breaks are escaped rather
	// than terminating the command and spilling the rest into the build log.
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != len(diffs) {
		t.Fatalf("expected %d annotations, got %d:\n%s", len(diffs), len(lines), output)
	}
	for i, line := range lines {
		if !strings.Contains(line, "%0A") {
			t.Errorf("annotation %d: expected the PEM's line breaks escaped, got: %s", i, line)
		}
	}

	// Still bounded: the bundle is past gitHubMaxValueLines and is truncated
	// like any other over-long value. Asserting the exact marker rather than
	// its "[" — every message already contains one, in the path itself.
	dropped := len(strings.Split(bundle, "\n")) - gitHubMaxValueLines
	marker := escapeGitHubData(fmt.Sprintf("[%d more lines]", dropped))
	if !strings.Contains(lines[2], marker) {
		t.Errorf("expected %q on the bundle annotation, got: %s", marker, lines[2])
	}
	if strings.Contains(lines[1], "more lines]") {
		t.Errorf("a single certificate is under the cap and must not be truncated, got: %s", lines[1])
	}
}

func TestGitHubFormatter_CertInspection_OneSidedFallsBackToDiff(t *testing.T) {
	// Mirrors the detailed formatter: a certificate replaced by something else
	// is a plain value change, so the line diff still applies.
	cert, _ := githubTestCerts(t)

	f := &GitHubFormatter{}
	msg := githubMessage(t, f.Format([]Difference{
		{Path: DiffPath{"data", "tls.crt"}, Type: DiffModified, From: cert, To: "not-a-cert"},
	}, DefaultFormatOptions()))

	if strings.Contains(msg, "Certificate(") {
		t.Errorf("expected no summary when only one side is a certificate, got: %s", msg)
	}
	if !strings.Contains(msg, "changed in multiline text") {
		t.Errorf("expected the line-diff path, got: %s", msg)
	}
}

func TestGithubDiffDescription_NilOptionsInspectsCerts(t *testing.T) {
	// Nil options must mean DefaultFormatOptions, so inspection is on. Reading
	// NoCertInspection off a nil-checked pointer would silently disable it.
	fromCert, toCert := githubTestCerts(t)
	diff := Difference{Path: DiffPath{"data", "tls.crt"}, Type: DiffModified, From: fromCert, To: toCert}

	got := githubDiffDescription(diff, nil)
	if want := githubDiffDescription(diff, DefaultFormatOptions()); got != want {
		t.Errorf("nil opts = %q, want %q", got, want)
	}
	if !strings.Contains(got, "Certificate(") {
		t.Errorf("expected nil opts to inspect certificates, got: %s", got)
	}
}

func TestGithubCertHelpers_PassThrough(t *testing.T) {
	cert, _ := githubTestCerts(t)

	t.Run("value: non-string", func(t *testing.T) {
		if got := githubCertValue(42, true); got != 42 {
			t.Errorf("expected 42 untouched, got %v", got)
		}
	})
	t.Run("value: non-cert string", func(t *testing.T) {
		if got := githubCertValue("plain", true); got != "plain" {
			t.Errorf("expected the string untouched, got %v", got)
		}
	})
	t.Run("value: disabled", func(t *testing.T) {
		if got := githubCertValue(cert, false); got != cert {
			t.Errorf("expected the PEM untouched when disabled")
		}
	})
	t.Run("pair: non-string side", func(t *testing.T) {
		from, to := githubCertPair(cert, 42, true)
		if from != cert || to != 42 {
			t.Errorf("expected both sides untouched, got %v / %v", from, to)
		}
	})
}

func TestGitHubFormatter_CertInspection_ChainIsNotSummarized(t *testing.T) {
	// A bundle is not a single certificate, so IsPEMCertificate rejects it —
	// but FormatCertificate would happily decode only its *first* block and
	// report the bundle as that one cert, silently hiding the rest. The
	// IsPEMCertificate guard is what stops that, on both the single-value and
	// the modified-pair path.
	first, second := githubTestCerts(t)
	chain := first + second

	f := &GitHubFormatter{}
	opts := DefaultFormatOptions()

	t.Run("added value", func(t *testing.T) {
		msg := githubMessage(t, f.Format([]Difference{
			{Path: DiffPath{"data", "ca-bundle.crt"}, Type: DiffAdded, To: chain},
		}, opts))

		if strings.Contains(msg, "Certificate(") {
			t.Errorf("expected a bundle to stay raw, not be reported as its first cert: %s", msg)
		}
		if !strings.Contains(msg, escapeGitHubData("BEGIN CERTIFICATE")) {
			t.Errorf("expected the raw bundle, got: %s", msg)
		}
	})

	t.Run("modified pair", func(t *testing.T) {
		// Only one side is a single certificate; the pair must not be
		// summarized on the strength of that side alone.
		msg := githubMessage(t, f.Format([]Difference{
			{Path: DiffPath{"data", "ca-bundle.crt"}, Type: DiffModified, From: chain, To: second},
		}, opts))

		if strings.Contains(msg, "Certificate(") {
			t.Errorf("expected no summary when one side is a bundle, got: %s", msg)
		}
	})

	t.Run("helpers directly", func(t *testing.T) {
		if got := githubCertValue(chain, true); got != chain {
			t.Errorf("githubCertValue summarized a bundle: %v", got)
		}
		if from, to := githubCertPair(chain, second, true); from != chain || to != second {
			t.Errorf("githubCertPair summarized a pair whose from side is a bundle")
		}
		if from, to := githubCertPair(second, chain, true); from != second || to != chain {
			t.Errorf("githubCertPair summarized a pair whose to side is a bundle")
		}
	})
}
