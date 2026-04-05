package profiles

import "strings"

// OIDCData holds the OIDC values derived from an Authentik provider
type OIDCData struct {
	ClientID     string
	ClientSecret string
	AuthorizeURL string
	TokenURL     string
	UserinfoURL  string
	IssuerURL    string
	LogoutURL    string
	Scopes       string
	RedirectURIs []string
}

// SigningCert holds the SAML signing certificate data from Authentik.
type SigningCert struct {
	CertificatePEM    string
	FingerprintSHA256 string
}

// BuildOIDCData constructs OIDCData from Authentik provider details and base URL.
//
// IMPORTANT: Token and Userinfo endpoints are GLOBAL in Authentik, not per-slug.
//   - Authorize: {baseURL}/application/o/authorize/
//   - Token:     {baseURL}/application/o/token/
//   - Userinfo:  {baseURL}/application/o/userinfo/
//   - Issuer:    {baseURL}/application/o/{slug}/  (per-slug)
//   - Logout:    {baseURL}/application/o/{slug}/end-session/  (per-slug)
func BuildOIDCData(baseURL, slug, clientID, clientSecret string, redirectURIs []string) OIDCData {
	baseURL = strings.TrimRight(baseURL, "/")
	return OIDCData{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthorizeURL: baseURL + "/application/o/authorize/",
		TokenURL:     baseURL + "/application/o/token/",
		UserinfoURL:  baseURL + "/application/o/userinfo/",
		IssuerURL:    baseURL + "/application/o/" + slug + "/",
		LogoutURL:    baseURL + "/application/o/" + slug + "/end-session/",
		Scopes:       "openid email profile",
		RedirectURIs: redirectURIs,
	}
}

// AppRootURL extracts the application root URL from the first redirect URI.
// e.g., "https://grafana.example.com/login/generic_oauth" -> "https://grafana.example.com/"
func (d OIDCData) AppRootURL() string {
	if len(d.RedirectURIs) == 0 {
		return ""
	}
	uri := d.RedirectURIs[0]
	// Find the third slash (after scheme://)
	schemeEnd := strings.Index(uri, "://")
	if schemeEnd < 0 {
		return ""
	}
	pathStart := strings.Index(uri[schemeEnd+3:], "/")
	if pathStart < 0 {
		return uri + "/"
	}
	return uri[:schemeEnd+3+pathStart] + "/"
}
