package authentik

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
)

// ParseCertificates accepts PEM, base64-encoded DER, or raw DER data
// and returns parsed certificates plus the PEM representation.
//
// Detection order:
//  1. Try PEM decode
//  2. Try base64 decode followed by DER parse
//  3. Try raw DER parse
func ParseCertificates(data []byte) ([]*x509.Certificate, []byte, error) {
	// 1. Try PEM decode
	if certs, pemOut, ok := parsePEM(data); ok {
		return certs, pemOut, nil
	}

	// 2. Try base64-encoded DER
	if decoded, err := base64.StdEncoding.DecodeString(string(data)); err == nil {
		if certs, err := x509.ParseCertificates(decoded); err == nil && len(certs) > 0 {
			pemOut := certsToAllPEM(certs)
			return certs, pemOut, nil
		}
	}

	// 3. Try raw DER
	if certs, err := x509.ParseCertificates(data); err == nil && len(certs) > 0 {
		pemOut := certsToAllPEM(certs)
		return certs, pemOut, nil
	}

	return nil, nil, fmt.Errorf("failed to parse certificates: data is not valid PEM, base64-encoded DER, or raw DER")
}

// CertFingerprint returns the SHA256 fingerprint of a certificate
// as a colon-separated uppercase hex string (e.g., "AB:CD:EF:...").
func CertFingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":")
}

// CertToPEM encodes DER bytes as a PEM CERTIFICATE block.
func CertToPEM(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
}

// PEMToFingerprint parses the first certificate from PEM data
// and returns its SHA256 fingerprint.
func PEMToFingerprint(pemData []byte) (string, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return "", fmt.Errorf("no PEM block found in data")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing certificate from PEM: %w", err)
	}

	return CertFingerprint(cert), nil
}

// parsePEM attempts to decode one or more PEM CERTIFICATE blocks from data.
// Returns the parsed certificates, the original PEM bytes, and whether parsing succeeded.
func parsePEM(data []byte) ([]*x509.Certificate, []byte, bool) {
	var certs []*x509.Certificate
	var pemOut []byte
	rest := data

	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		certs = append(certs, cert)
		pemOut = append(pemOut, pem.EncodeToMemory(block)...)
	}

	if len(certs) == 0 {
		return nil, nil, false
	}
	return certs, pemOut, true
}

// certsToAllPEM encodes all certificates as PEM blocks.
func certsToAllPEM(certs []*x509.Certificate) []byte {
	pemOut := make([]byte, 0, len(certs)*512)
	for _, cert := range certs {
		pemOut = append(pemOut, CertToPEM(cert.Raw)...)
	}
	return pemOut
}
