package authentik

// ApplicationResponse represents an Authentik application looked up by slug.
// PK is a UUID string in Authentik (not an integer).
type ApplicationResponse struct {
	PK   string `json:"pk"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// OAuth2Provider represents an Authentik OAuth2/OIDC provider
type OAuth2Provider struct {
	PK                      int    `json:"pk"`
	Name                    string `json:"name"`
	ClientID                string `json:"client_id"`
	ClientSecret            string `json:"client_secret"`
	ClientType              string `json:"client_type"`
	AssignedApplicationSlug string `json:"assigned_application_slug"`
	AssignedApplicationName string `json:"assigned_application_name"`
	RedirectURIs            string `json:"redirect_uris"`
}

// Pagination holds Authentik API pagination metadata
type Pagination struct {
	Count      int     `json:"count"`
	Next       *string `json:"next"`
	TotalPages int     `json:"total_pages"`
}

// ProviderListResponse is the paginated response from the providers endpoint
type ProviderListResponse struct {
	Pagination Pagination      `json:"pagination"`
	Results    []OAuth2Provider `json:"results"`
}

// TokenCreateResponse is the response when creating an API token
// Note: does NOT include key — use view_key endpoint separately
type TokenCreateResponse struct {
	PK         string `json:"pk"`
	Identifier string `json:"identifier"`
}

// TokenViewKeyResponse is the response from the view_key endpoint
type TokenViewKeyResponse struct {
	Key string `json:"key"`
}
