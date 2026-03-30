package authentik

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetOAuth2ProviderBySlug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v3/core/applications/grafana/":
			// Step 1: Get application by slug
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			resp := ApplicationResponse{
				PK:   "a1b2c3d4-0001-0000-0000-000000000001",
				Slug: "grafana",
				Name: "Grafana",
			}
			_ = json.NewEncoder(w).Encode(resp)

		case "/api/v3/providers/oauth2/":
			// Step 2: Get providers filtered by application PK
			if r.URL.Query().Get("application") != "a1b2c3d4-0001-0000-0000-000000000001" {
				t.Errorf("expected application=a1b2c3d4-0001-0000-0000-000000000001, got %s", r.URL.Query().Get("application"))
			}
			resp := ProviderListResponse{
				Pagination: Pagination{Count: 1},
				Results: []OAuth2Provider{
					{
						PK:                      1,
						Name:                    "grafana-oidc",
						ClientID:                "client-id-123",
						ClientSecret:            "client-secret-456",
						ClientType:              "confidential",
						SigningKey:              stringPtr("uuid-signing-key-456"),
						AssignedApplicationSlug: "grafana",
						AssignedApplicationName: "Grafana",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	provider, err := c.GetOAuth2ProviderBySlug(context.Background(), "grafana")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.ClientID != "client-id-123" {
		t.Errorf("expected client-id-123, got %s", provider.ClientID)
	}
	if provider.ClientSecret != "client-secret-456" {
		t.Errorf("expected client-secret-456, got %s", provider.ClientSecret)
	}
	if provider.SigningKey == nil || *provider.SigningKey != "uuid-signing-key-456" {
		t.Errorf("expected signing key uuid-signing-key-456, got %v", provider.SigningKey)
	}
}

func TestGetOAuth2ProviderBySlug_AppNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/core/applications/nonexistent/" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detail": "Not found."}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	_, err := c.GetOAuth2ProviderBySlug(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing application")
	}
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestGetOAuth2ProviderBySlug_NoProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v3/core/applications/grafana/":
			resp := ApplicationResponse{PK: "a1b2c3d4-0001-0000-0000-000000000001", Slug: "grafana", Name: "Grafana"}
			_ = json.NewEncoder(w).Encode(resp)
		case "/api/v3/providers/oauth2/":
			resp := ProviderListResponse{Pagination: Pagination{Count: 0}, Results: []OAuth2Provider{}}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	_, err := c.GetOAuth2ProviderBySlug(context.Background(), "grafana")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestGetOAuth2ProviderBySlug_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := NewClient(server.URL, "bad-token")
	_, err := c.GetOAuth2ProviderBySlug(context.Background(), "grafana")
	if err == nil {
		t.Fatal("expected error for unauthorized")
	}
}

func TestGetOAuth2ProviderBySlug_TokenExpired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"detail":"Token invalid/expired"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "expired-token")
	_, err := c.GetOAuth2ProviderBySlug(context.Background(), "test-app")
	if err == nil {
		t.Fatal("expected error for forbidden response")
	}
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestClearToken(t *testing.T) {
	c := NewClient("http://localhost", "initial-token")
	if !c.HasToken() {
		t.Fatal("expected HasToken() to be true after NewClient with token")
	}
	c.ClearToken()
	if c.HasToken() {
		t.Fatal("expected HasToken() to be false after ClearToken()")
	}
}

func TestCreateAPIToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Verify Bearer token auth (not Basic auth — Authentik does not support Basic)
		if !hasBearer(r) {
			t.Errorf("expected Bearer auth, got: %s", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case r.URL.Path == "/api/v3/core/tokens/" && r.Method == http.MethodPost:
			// Token creation — note: does NOT return key
			resp := TokenCreateResponse{
				PK:         "uuid-123",
				Identifier: "authentik-operator",
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(resp)

		case r.URL.Path == "/api/v3/core/tokens/authentik-operator/view_key/" && r.Method == http.MethodGet:
			// Separate call to retrieve the actual key
			resp := TokenViewKeyResponse{
				Key: "generated-api-key-789",
			}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "bootstrap-token")
	key, err := c.CreateAPIToken(context.Background(), "authentik-operator")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "generated-api-key-789" {
		t.Errorf("expected generated-api-key-789, got %s", key)
	}
}

func TestCreateAPIToken_AlreadyExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v3/core/tokens/" && r.Method == http.MethodPost:
			// 400 — token with this identifier already exists
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"identifier": ["token with this identifier already exists."]}`))

		case r.URL.Path == "/api/v3/core/tokens/authentik-operator/view_key/" && r.Method == http.MethodGet:
			// Fall back to retrieving existing token's key
			resp := TokenViewKeyResponse{Key: "existing-key-456"}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "bootstrap-token")
	key, err := c.CreateAPIToken(context.Background(), "authentik-operator")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "existing-key-456" {
		t.Errorf("expected existing-key-456, got %s", key)
	}
}

func TestNewClient_WithInsecureSkipVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Without insecure — should fail TLS
	c := NewClient(server.URL, "test-token")
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	_, err := c.httpClient.Do(req)
	if err == nil {
		t.Fatal("expected TLS error without insecure skip verify")
	}

	// With insecure — should succeed
	c2 := NewClient(server.URL, "test-token", WithInsecureSkipVerify(true))
	req2, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	resp, err := c2.httpClient.Do(req2)
	if err != nil {
		t.Fatalf("unexpected error with insecure skip verify: %v", err)
	}
	resp.Body.Close()
}

func TestNewClient_WithCACertData(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverCert := server.TLS.Certificates[0]
	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCert.Certificate[0],
	})

	c := NewClient(server.URL, "test-token", WithCACertData(caPEM))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error with CA cert: %v", err)
	}
	resp.Body.Close()
}

func TestNewClient_WithCACertPath(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverCert := server.TLS.Certificates[0]
	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCert.Certificate[0],
	})
	tmpFile := t.TempDir() + "/ca.crt"
	if err := os.WriteFile(tmpFile, caPEM, 0644); err != nil {
		t.Fatalf("writing temp cert: %v", err)
	}

	c := NewClient(server.URL, "test-token", WithCACertPath(tmpFile))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error with CA cert path: %v", err)
	}
	resp.Body.Close()
}

func TestGetCertificateByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/api/v3/crypto/certificatekeypairs/uuid-cert-123/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		resp := map[string]interface{}{
			"pk":                 "uuid-cert-123",
			"name":               "authentik Self-signed Certificate",
			"certificate_data":   "-----BEGIN CERTIFICATE-----\nMIIBtest\n-----END CERTIFICATE-----\n",
			"fingerprint_sha256": "AB:CD:EF:01:23:45:67:89",
			"cert_expiry":        "2027-01-01T00:00:00Z",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	cert, err := c.GetCertificateByID(context.Background(), "uuid-cert-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert.Name != "authentik Self-signed Certificate" {
		t.Errorf("expected cert name, got %s", cert.Name)
	}
	if cert.FingerprintSHA256 != "AB:CD:EF:01:23:45:67:89" {
		t.Errorf("expected fingerprint, got %s", cert.FingerprintSHA256)
	}
	if cert.CertificateData == "" {
		t.Error("expected certificate PEM data")
	}
}

func TestGetCertificateByID_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail": "Not found."}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	_, err := c.GetCertificateByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing cert")
	}
}

func stringPtr(s string) *string { return &s }

func hasBearer(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	return len(auth) > 7 && auth[:7] == "Bearer "
}
