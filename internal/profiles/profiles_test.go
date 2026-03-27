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
		LogoutURL:    "https://auth.example.com/application/o/test-app/end-session/",
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
	// Logout URL is per-slug
	if data.LogoutURL != "https://auth.example.com/application/o/test-app/end-session/" {
		t.Errorf("logout URL should be per-slug end-session, got %s", data.LogoutURL)
	}
}

func TestGrafanaProfile(t *testing.T) {
	data := newTestData()
	result := Apply("grafana", data, nil)

	checks := map[string]string{
		"GF_AUTH_GENERIC_OAUTH_ENABLED":                    "true",
		"GF_AUTH_GENERIC_OAUTH_NAME":                       "authentik",
		"GF_AUTH_GENERIC_OAUTH_CLIENT_ID":                  "test-client-id",
		"GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET":              "test-client-secret",
		"GF_AUTH_GENERIC_OAUTH_AUTH_URL":                   "https://auth.example.com/application/o/authorize/",
		"GF_AUTH_GENERIC_OAUTH_TOKEN_URL":                  "https://auth.example.com/application/o/token/",
		"GF_AUTH_GENERIC_OAUTH_API_URL":                    "https://auth.example.com/application/o/userinfo/",
		"GF_AUTH_GENERIC_OAUTH_SCOPES":                     "openid email profile",
		"GF_AUTH_SIGNOUT_REDIRECT_URL":                     "https://auth.example.com/application/o/test-app/end-session/",
		"GF_AUTH_OAUTH_AUTO_LOGIN":                         "true",
		"GF_AUTH_GENERIC_OAUTH_ALLOW_ASSIGN_GRAFANA_ADMIN": "true",
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

func TestRagFlowProfile(t *testing.T) {
	data := newTestData()
	overrides := map[string]string{
		"redirect_uri": "https://ragflow.example.com/v1/user/oauth/callback/oidc",
	}
	result := Apply("ragflow", data, overrides)

	// Check individual keys
	checks := map[string]string{
		"client_id":     "test-client-id",
		"client_secret": "test-client-secret",
		"issuer":        "https://auth.example.com/application/o/test-app/",
		"scope":         "openid email profile",
	}

	for key, expected := range checks {
		if result[key] != expected {
			t.Errorf("ragflow profile: key %s = %q, want %q", key, result[key], expected)
		}
	}

	// Check service_conf_yaml contains key values
	yaml := result["service_conf_yaml"]
	if yaml == "" {
		t.Fatal("ragflow profile: service_conf_yaml is empty")
	}
	for _, needle := range []string{"client_id: \"test-client-id\"", "client_secret: \"test-client-secret\"", "redirect_uri: \"https://ragflow.example.com"} {
		if !contains(yaml, needle) {
			t.Errorf("ragflow profile: service_conf_yaml missing %q", needle)
		}
	}

	// redirect_uri should NOT leak into the secret as a separate key
	if _, ok := result["redirect_uri"]; ok {
		t.Error("ragflow profile: redirect_uri should be removed from secret keys")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestHarborProfile(t *testing.T) {
	data := newTestData()
	result := Apply("harbor", data, nil)

	checks := map[string]string{
		"client_id":     "test-client-id",
		"client_secret": "test-client-secret",
		"issuer":        "https://auth.example.com/application/o/test-app/",
		"scopes":        "openid email profile",
		"authorize_url": "https://auth.example.com/application/o/authorize/",
		"token_url":     "https://auth.example.com/application/o/token/",
		"userinfo_url":  "https://auth.example.com/application/o/userinfo/",
	}

	for key, expected := range checks {
		if result[key] != expected {
			t.Errorf("harbor profile: key %s = %q, want %q", key, result[key], expected)
		}
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

func TestTemplateOverrides(t *testing.T) {
	data := newTestData()
	overrides := map[string]string{
		"social_auth_oidc_key":    "{{.ClientID}}",
		"social_auth_oidc_secret": "{{.ClientSecret}}",
		"oidc_endpoint":           "{{.IssuerURL}}",
		"static_value":            "no-template-here",
	}
	result := Apply("generic", data, overrides)

	checks := map[string]string{
		"social_auth_oidc_key":    "test-client-id",
		"social_auth_oidc_secret": "test-client-secret",
		"oidc_endpoint":           "https://auth.example.com/application/o/test-app/",
		"static_value":            "no-template-here",
		// Original generic keys should still be present
		"clientId":     "test-client-id",
		"clientSecret": "test-client-secret",
	}

	for key, expected := range checks {
		if result[key] != expected {
			t.Errorf("template overrides: key %s = %q, want %q", key, result[key], expected)
		}
	}
}

func TestArgoCDProfile_WithSigningCert(t *testing.T) {
	data := newTestData()
	cert := &SigningCert{
		CertificatePEM:    "-----BEGIN CERTIFICATE-----\nMIIBtest\n-----END CERTIFICATE-----\n",
		FingerprintSHA256: "AB:CD:EF:01:23:45",
	}
	result := ApplyWithCert("argocd", data, nil, cert)

	if result["dex.authentik.clientSecret"] != data.ClientSecret {
		t.Error("missing clientSecret")
	}
	if result["caData"] == "" {
		t.Error("argocd profile should include caData when signing cert provided")
	}
	if result["caFingerprint"] != "AB:CD:EF:01:23:45" {
		t.Errorf("expected fingerprint, got %s", result["caFingerprint"])
	}
}

func TestGenericProfile_WithSigningCert(t *testing.T) {
	data := newTestData()
	cert := &SigningCert{
		CertificatePEM:    "-----BEGIN CERTIFICATE-----\nMIIBtest\n-----END CERTIFICATE-----\n",
		FingerprintSHA256: "AB:CD:EF:01:23:45",
	}
	result := ApplyWithCert("generic", data, nil, cert)

	if result["saml.crt"] != cert.CertificatePEM {
		t.Error("generic profile should include saml.crt")
	}
	if result["saml.fingerprint"] != "AB:CD:EF:01:23:45" {
		t.Error("generic profile should include saml.fingerprint")
	}
}

func TestApplyWithCert_NilCert(t *testing.T) {
	data := newTestData()
	result := ApplyWithCert("argocd", data, nil, nil)

	if result["dex.authentik.clientSecret"] != data.ClientSecret {
		t.Error("missing clientSecret")
	}
	if _, ok := result["caData"]; ok {
		t.Error("should not include caData when cert is nil")
	}
}

func TestGrafanaProfile_WithSigningCert(t *testing.T) {
	data := newTestData()
	cert := &SigningCert{
		CertificatePEM:    "-----BEGIN CERTIFICATE-----\nMIIBtest\n-----END CERTIFICATE-----\n",
		FingerprintSHA256: "AB:CD:EF:01:23:45",
	}
	result := ApplyWithCert("grafana", data, nil, cert)

	if result["saml.crt"] != cert.CertificatePEM {
		t.Error("should include saml.crt for profiles that don't have special handling")
	}
}

func TestTemplateOverrideInvalidTemplate(t *testing.T) {
	data := newTestData()
	overrides := map[string]string{
		"bad_template": "{{.NonExistentField}}",
		"good_key":     "{{.ClientID}}",
	}
	result := Apply("generic", data, overrides)

	// Invalid template should keep original value
	if result["bad_template"] != "{{.NonExistentField}}" {
		t.Errorf("invalid template should keep original, got %q", result["bad_template"])
	}
	// Valid template should resolve
	if result["good_key"] != "test-client-id" {
		t.Errorf("valid template should resolve, got %q", result["good_key"])
	}
}
