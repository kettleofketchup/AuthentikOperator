package authentik

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var ErrProviderNotFound = errors.New("oauth2 provider not found for application slug")

// Client is an HTTP client for the Authentik API v3
type Client struct {
	baseURL    string
	token      string
	mu         sync.RWMutex
	httpClient *http.Client
}

// NewClient creates a new Authentik API client.
// All requests use Bearer token auth (Authentik does not support Basic auth).
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetToken updates the Bearer token (used when refreshing from K8s secret).
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// GetOAuth2ProviderBySlug fetches the OAuth2 provider for a given application slug.
// This is a two-step process:
//  1. GET /api/v3/core/applications/{slug}/ to get the application PK
//  2. GET /api/v3/providers/oauth2/?application={pk} to get the provider
//
// The application__slug filter does NOT exist on the providers endpoint.
func (c *Client) GetOAuth2ProviderBySlug(ctx context.Context, slug string) (*OAuth2Provider, error) {
	// Step 1: Get application by slug
	appURL := fmt.Sprintf("%s/api/v3/core/applications/%s/", c.baseURL, url.PathEscape(slug))
	appReq, err := http.NewRequestWithContext(ctx, http.MethodGet, appURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating application request: %w", err)
	}
	c.setAuth(appReq)

	appResp, err := c.httpClient.Do(appReq)
	if err != nil {
		return nil, fmt.Errorf("fetching application: %w", err)
	}
	defer appResp.Body.Close() //nolint:errcheck

	if appResp.StatusCode == http.StatusNotFound {
		return nil, ErrProviderNotFound
	}
	if appResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(appResp.Body, 1<<20))
		return nil, fmt.Errorf("authentik API returned %d for application %q: %s", appResp.StatusCode, slug, string(body))
	}

	var app ApplicationResponse
	if err := json.NewDecoder(appResp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("decoding application response: %w", err)
	}
	io.Copy(io.Discard, appResp.Body) //nolint:errcheck

	// Step 2: Get providers filtered by application PK
	provURL := fmt.Sprintf("%s/api/v3/providers/oauth2/?application=%s", c.baseURL, app.PK)
	provReq, err := http.NewRequestWithContext(ctx, http.MethodGet, provURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating provider request: %w", err)
	}
	c.setAuth(provReq)

	provResp, err := c.httpClient.Do(provReq)
	if err != nil {
		return nil, fmt.Errorf("fetching providers: %w", err)
	}
	defer provResp.Body.Close() //nolint:errcheck

	if provResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(provResp.Body, 1<<20))
		return nil, fmt.Errorf("authentik API returned %d for providers: %s", provResp.StatusCode, string(body))
	}

	var listResp ProviderListResponse
	if err := json.NewDecoder(provResp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("decoding provider response: %w", err)
	}
	io.Copy(io.Discard, provResp.Body) //nolint:errcheck

	if len(listResp.Results) == 0 {
		return nil, ErrProviderNotFound
	}

	return &listResp.Results[0], nil
}

// tokenCreateRequest is the JSON body for POST /api/v3/core/tokens/.
type tokenCreateRequest struct {
	Identifier  string `json:"identifier"`
	Intent      string `json:"intent"`
	Description string `json:"description"`
}

// CreateAPIToken creates an API token in Authentik using Bearer auth.
// If a token with the given identifier already exists, retrieves its key instead.
//
// Flow:
//  1. POST /api/v3/core/tokens/ to create (may 400 if exists)
//  2. GET /api/v3/core/tokens/{identifier}/view_key/ to retrieve the key
func (c *Client) CreateAPIToken(ctx context.Context, identifier string) (string, error) {
	reqBody, err := json.Marshal(tokenCreateRequest{
		Identifier:  identifier,
		Intent:      "api",
		Description: "AuthentikOperator service token",
	})
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	createURL := fmt.Sprintf("%s/api/v3/core/tokens/", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		// Token was created successfully; retrieve its key.
	case http.StatusBadRequest:
		// Read the body to determine whether this is an "already exists" conflict
		// or a genuine validation error we should surface.
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if !strings.Contains(string(respBody), "already exists") {
			return "", fmt.Errorf("authentik API returned 400: %s", string(respBody))
		}
		// Token already exists — fall through to view_key retrieval.
	default:
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("authentik API returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Retrieve the key via the view_key endpoint
	return c.viewTokenKey(ctx, identifier)
}

// viewTokenKey retrieves the key for an existing token.
func (c *Client) viewTokenKey(ctx context.Context, identifier string) (string, error) {
	viewURL := fmt.Sprintf("%s/api/v3/core/tokens/%s/view_key/", c.baseURL, url.PathEscape(identifier))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, viewURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating view_key request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching token key: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("view_key returned %d: %s", resp.StatusCode, string(body))
	}

	var keyResp TokenViewKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		return "", fmt.Errorf("decoding view_key response: %w", err)
	}

	return keyResp.Key, nil
}

func (c *Client) setAuth(req *http.Request) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
}
