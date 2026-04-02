package authentik

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

const (
	testCertCN         = "test-cert"
	pemTypeCertificate = "CERTIFICATE"
)

// newSelfSignedCert generates an ECDSA P256 self-signed certificate for testing.
// It returns the certificate in PEM and raw DER formats.
func newSelfSignedCert(t *testing.T) (pemBytes, derBytes []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating ECDSA key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: testCertCN},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	derBytes, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating self-signed certificate: %v", err)
	}

	pemBytes = pem.EncodeToMemory(&pem.Block{
		Type:  pemTypeCertificate,
		Bytes: derBytes,
	})

	return pemBytes, derBytes
}

func TestParseCertificates_PEM(t *testing.T) {
	pemBytes, _ := newSelfSignedCert(t)

	certs, pemOut, err := ParseCertificates(pemBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(certs) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(certs))
	}
	if certs[0].Subject.CommonName != testCertCN {
		t.Errorf("expected CN=test-cert, got %s", certs[0].Subject.CommonName)
	}
	if len(pemOut) == 0 {
		t.Error("expected non-empty PEM output")
	}

	// Verify the returned PEM is valid
	block, _ := pem.Decode(pemOut)
	if block == nil {
		t.Fatal("returned PEM data could not be decoded")
	}
	if block.Type != pemTypeCertificate {
		t.Errorf("expected CERTIFICATE block type, got %s", block.Type)
	}
}

func TestParseCertificates_Base64DER(t *testing.T) {
	_, derBytes := newSelfSignedCert(t)

	b64 := make([]byte, base64.StdEncoding.EncodedLen(len(derBytes)))
	base64.StdEncoding.Encode(b64, derBytes)

	certs, pemOut, err := ParseCertificates(b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(certs) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(certs))
	}
	if certs[0].Subject.CommonName != testCertCN {
		t.Errorf("expected CN=test-cert, got %s", certs[0].Subject.CommonName)
	}
	if len(pemOut) == 0 {
		t.Error("expected non-empty PEM output")
	}
}

func TestParseCertificates_RawDER(t *testing.T) {
	_, derBytes := newSelfSignedCert(t)

	certs, pemOut, err := ParseCertificates(derBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(certs) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(certs))
	}
	if certs[0].Subject.CommonName != testCertCN {
		t.Errorf("expected CN=test-cert, got %s", certs[0].Subject.CommonName)
	}
	if len(pemOut) == 0 {
		t.Error("expected non-empty PEM output")
	}
}

func TestParseCertificates_Invalid(t *testing.T) {
	_, _, err := ParseCertificates([]byte("this is not a certificate"))
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestCertFingerprint(t *testing.T) {
	pemBytes, _ := newSelfSignedCert(t)
	certs, _, err := ParseCertificates(pemBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fp := CertFingerprint(certs[0])

	// SHA256 fingerprint: 64 hex chars + 31 colons = 95 chars
	if len(fp) != 95 {
		t.Errorf("expected fingerprint length 95, got %d: %s", len(fp), fp)
	}

	// Verify colon-separated hex format
	parts := 0
	for i, c := range fp {
		if (i+1)%3 == 0 {
			if c != ':' {
				t.Errorf("expected colon at position %d, got %c", i, c)
			}
		} else {
			if (c < '0' || c > '9') && (c < 'A' || c > 'F') {
				t.Errorf("expected uppercase hex digit at position %d, got %c", i, c)
			}
			parts++
		}
	}
}

func TestCertToPEM(t *testing.T) {
	_, derBytes := newSelfSignedCert(t)

	pemOut := CertToPEM(derBytes)
	if len(pemOut) == 0 {
		t.Fatal("expected non-empty PEM output")
	}

	// Verify roundtrip: PEM decode back to DER should match
	block, rest := pem.Decode(pemOut)
	if block == nil {
		t.Fatal("PEM decode failed")
	}
	if block.Type != pemTypeCertificate {
		t.Errorf("expected CERTIFICATE block, got %s", block.Type)
	}
	if len(rest) != 0 {
		t.Errorf("expected no trailing data, got %d bytes", len(rest))
	}

	// Verify DER bytes match
	if len(block.Bytes) != len(derBytes) {
		t.Fatalf("DER length mismatch: got %d, want %d", len(block.Bytes), len(derBytes))
	}
	for i := range derBytes {
		if block.Bytes[i] != derBytes[i] {
			t.Fatalf("DER byte mismatch at index %d", i)
			break
		}
	}
}

func TestPEMToFingerprint(t *testing.T) {
	pemBytes, _ := newSelfSignedCert(t)

	fp, err := PEMToFingerprint(pemBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it matches CertFingerprint for the same cert
	certs, _, err := ParseCertificates(pemBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := CertFingerprint(certs[0])
	if fp != expected {
		t.Errorf("fingerprint mismatch: got %s, want %s", fp, expected)
	}
}

func TestPEMToFingerprint_Invalid(t *testing.T) {
	_, err := PEMToFingerprint([]byte("not valid PEM data"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}
