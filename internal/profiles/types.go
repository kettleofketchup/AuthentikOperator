package profiles

// OIDCData holds the OIDC values derived from an Authentik provider
type OIDCData struct {
	ClientID     string
	ClientSecret string
	AuthorizeURL string
	TokenURL     string
	UserinfoURL  string
	IssuerURL    string
	Scopes       string
}

// BuildOIDCData constructs OIDCData from Authentik provider details and base URL.
//
// IMPORTANT: Token and Userinfo endpoints are GLOBAL in Authentik, not per-slug.
//   - Authorize: {baseURL}/application/o/authorize/
//   - Token:     {baseURL}/application/o/token/
//   - Userinfo:  {baseURL}/application/o/userinfo/
//   - Issuer:    {baseURL}/application/o/{slug}/  (per-slug)
func BuildOIDCData(baseURL, slug, clientID, clientSecret string) OIDCData {
	return OIDCData{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthorizeURL: baseURL + "/application/o/authorize/",
		TokenURL:     baseURL + "/application/o/token/",
		UserinfoURL:  baseURL + "/application/o/userinfo/",
		IssuerURL:    baseURL + "/application/o/" + slug + "/",
		Scopes:       "openid email profile",
	}
}
