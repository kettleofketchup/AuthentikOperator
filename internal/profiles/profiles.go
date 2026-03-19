package profiles

import "maps"

// Apply maps OIDCData to a secret data map using the named profile,
// then merges overrides on top.
func Apply(profileName string, data OIDCData, overrides map[string]string) map[string]string {
	var result map[string]string

	switch profileName {
	case "grafana":
		result = grafana(data)
	case "openwebui":
		result = openwebui(data)
	case "argocd":
		result = argocd(data)
	default:
		result = generic(data)
	}

	maps.Copy(result, overrides)

	return result
}

func grafana(data OIDCData) map[string]string {
	return map[string]string{
		"GF_AUTH_GENERIC_OAUTH_ENABLED":       "true",
		"GF_AUTH_GENERIC_OAUTH_NAME":          "authentik",
		"GF_AUTH_GENERIC_OAUTH_CLIENT_ID":     data.ClientID,
		"GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET": data.ClientSecret,
		"GF_AUTH_GENERIC_OAUTH_AUTH_URL":      data.AuthorizeURL,
		"GF_AUTH_GENERIC_OAUTH_TOKEN_URL":     data.TokenURL,
		"GF_AUTH_GENERIC_OAUTH_API_URL":       data.UserinfoURL,
		"GF_AUTH_GENERIC_OAUTH_SCOPES":        data.Scopes,
	}
}

func openwebui(data OIDCData) map[string]string {
	return map[string]string{
		"ENABLE_OAUTH_SIGNUP": "true",
		"OAUTH_PROVIDER_NAME": "authentik",
		"OAUTH_CLIENT_ID":     data.ClientID,
		"OAUTH_CLIENT_SECRET": data.ClientSecret,
		"OPENID_PROVIDER_URL": data.IssuerURL,
		"OAUTH_SCOPES":        data.Scopes,
	}
}

// argocd produces keys for the argocd-secret Secret.
// The OIDCClient CR for ArgoCD should target argocd-secret in the argocd namespace.
//
// ArgoCD's Dex integration requires:
//   - argocd-secret: key "dex.authentik.clientSecret" with the actual secret
//   - argocd-cm ConfigMap: key "dex.config" with the Dex connector YAML
//
// The dex.config in argocd-cm is NOT managed by this operator — configure it
// in your ArgoCD Helm values (server.config.dex\.config) with:
//
//	connectors:
//	- type: oidc
//	  id: authentik
//	  name: authentik
//	  config:
//	    issuer: https://auth.example.com/application/o/argocd/
//	    clientID: <from clientId key below>
//	    clientSecret: $dex.authentik.clientSecret
//	    insecureEnableGroups: true
//	    scopes: [openid, profile, email]
func argocd(data OIDCData) map[string]string {
	return map[string]string{
		"dex.authentik.clientSecret": data.ClientSecret,
		"clientId":                   data.ClientID,
		"issuerUrl":                  data.IssuerURL,
	}
}

func generic(data OIDCData) map[string]string {
	return map[string]string{
		"clientId":     data.ClientID,
		"clientSecret": data.ClientSecret,
		"authorizeUrl": data.AuthorizeURL,
		"tokenUrl":     data.TokenURL,
		"userinfoUrl":  data.UserinfoURL,
		"issuerUrl":    data.IssuerURL,
		"scopes":       data.Scopes,
	}
}
