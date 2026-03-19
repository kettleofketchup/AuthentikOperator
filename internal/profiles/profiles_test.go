package profiles

import (
	"testing"
)

func newTestData() OIDCData {
	return OIDCData{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		AuthorizeURL: "https://auth.example.com/application/o/authorize/",
		TokenURL:     "https://auth.example.com/application/o/token/",
		UserinfoURL:  "https://auth.example.com/application/o/userinfo/",
		IssuerURL:    "https://auth.example.com/application/o/test-app/",
		Scopes:       "openid email profile",
	}
}

func TestBuildOIDCData(t *testing.T) {
	data := BuildOIDCData("https://auth.example.com", "test-app", "cid", "csecret")

	// Token and userinfo URLs are GLOBAL (not per-slug)
	if data.TokenURL != "https://auth.example.com/application/o/token/" {
		t.Errorf("token URL should be global, got %s", data.TokenURL)
	}
	if data.UserinfoURL != "https://auth.example.com/application/o/userinfo/" {
		t.Errorf("userinfo URL should be global, got %s", data.UserinfoURL)
	}
	// Authorize URL is global
	if data.AuthorizeURL != "https://auth.example.com/application/o/authorize/" {
		t.Errorf("authorize URL should be global, got %s", data.AuthorizeURL)
	}
	// Issuer URL is per-slug
	if data.IssuerURL != "https://auth.example.com/application/o/test-app/" {
		t.Errorf("issuer URL should be per-slug, got %s", data.IssuerURL)
	}
}

func TestGrafanaProfile(t *testing.T) {
	data := newTestData()
	result := Apply("grafana", data, nil)

	checks := map[string]string{
		"GF_AUTH_GENERIC_OAUTH_ENABLED":       "true",
		"GF_AUTH_GENERIC_OAUTH_NAME":          "authentik",
		"GF_AUTH_GENERIC_OAUTH_CLIENT_ID":     "test-client-id",
		"GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET": "test-client-secret",
		"GF_AUTH_GENERIC_OAUTH_AUTH_URL":      "https://auth.example.com/application/o/authorize/",
		"GF_AUTH_GENERIC_OAUTH_TOKEN_URL":     "https://auth.example.com/application/o/token/",
		"GF_AUTH_GENERIC_OAUTH_API_URL":       "https://auth.example.com/application/o/userinfo/",
		"GF_AUTH_GENERIC_OAUTH_SCOPES":        "openid email profile",
	}

	for key, expected := range checks {
		if result[key] != expected {
			t.Errorf("grafana profile: key %s = %q, want %q", key, result[key], expected)
		}
	}
}

func TestOpenWebUIProfile(t *testing.T) {
	data := newTestData()
	result := Apply("openwebui", data, nil)

	checks := map[string]string{
		"ENABLE_OAUTH_SIGNUP": "true",
		"OAUTH_PROVIDER_NAME": "authentik",
		"OAUTH_CLIENT_ID":     "test-client-id",
		"OAUTH_CLIENT_SECRET": "test-client-secret",
		"OPENID_PROVIDER_URL": "https://auth.example.com/application/o/test-app/",
		"OAUTH_SCOPES":        "openid email profile",
	}

	for key, expected := range checks {
		if result[key] != expected {
			t.Errorf("openwebui profile: key %s = %q, want %q", key, result[key], expected)
		}
	}
}

func TestArgoCDProfile(t *testing.T) {
	data := newTestData()
	result := Apply("argocd", data, nil)

	// ArgoCD expects dex.authentik.clientSecret in argocd-secret Secret
	// and dex.config in argocd-cm ConfigMap (managed separately by ArgoCD Helm values)
	secret, ok := result["dex.authentik.clientSecret"]
	if !ok {
		t.Fatal("argocd profile: missing dex.authentik.clientSecret key")
	}
	if secret != data.ClientSecret {
		t.Errorf("argocd profile: dex.authentik.clientSecret = %q, want %q", secret, data.ClientSecret)
	}

	// Convenience keys so users can reference them in dex.config
	if result["clientId"] != data.ClientID {
		t.Errorf("argocd profile: clientId = %q, want %q", result["clientId"], data.ClientID)
	}
	if result["issuerUrl"] != data.IssuerURL {
		t.Errorf("argocd profile: issuerUrl = %q, want %q", result["issuerUrl"], data.IssuerURL)
	}
}

func TestGenericProfile(t *testing.T) {
	data := newTestData()
	result := Apply("generic", data, nil)

	checks := map[string]string{
		"clientId":     "test-client-id",
		"clientSecret": "test-client-secret",
		"authorizeUrl": "https://auth.example.com/application/o/authorize/",
		"tokenUrl":     "https://auth.example.com/application/o/token/",
		"userinfoUrl":  "https://auth.example.com/application/o/userinfo/",
		"issuerUrl":    "https://auth.example.com/application/o/test-app/",
		"scopes":       "openid email profile",
	}

	for key, expected := range checks {
		if result[key] != expected {
			t.Errorf("generic profile: key %s = %q, want %q", key, result[key], expected)
		}
	}
}

func TestOverrides(t *testing.T) {
	data := newTestData()
	overrides := map[string]string{
		"GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH": "contains(groups, 'admins') && 'Admin'",
		"GF_AUTH_GENERIC_OAUTH_SCOPES":              "openid email profile groups",
	}
	result := Apply("grafana", data, overrides)

	if result["GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH"] != "contains(groups, 'admins') && 'Admin'" {
		t.Error("override not applied for ROLE_ATTRIBUTE_PATH")
	}
	if result["GF_AUTH_GENERIC_OAUTH_SCOPES"] != "openid email profile groups" {
		t.Error("override did not replace default scopes")
	}
	if result["GF_AUTH_GENERIC_OAUTH_CLIENT_ID"] != "test-client-id" {
		t.Error("non-overridden key missing")
	}
}

func TestUnknownProfileFallsBackToGeneric(t *testing.T) {
	data := newTestData()
	result := Apply("nonexistent", data, nil)

	if result["clientId"] != "test-client-id" {
		t.Error("unknown profile should fall back to generic")
	}
}
