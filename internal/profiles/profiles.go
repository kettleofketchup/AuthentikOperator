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
	case "ragflow":
		result = ragflow(data, overrides)
	case "harbor":
		result = harbor(data)
	default:
		result = generic(data)
	}

	maps.Copy(result, overrides)
	// Remove internal-only keys that shouldn't appear in the secret
	delete(result, "redirect_uri")

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
		"GF_AUTH_SIGNOUT_REDIRECT_URL":        data.LogoutURL,
		"GF_AUTH_OAUTH_AUTO_LOGIN":            "true",
		"GF_AUTH_GENERIC_OAUTH_ALLOW_ASSIGN_GRAFANA_ADMIN": "true",
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

// ragflow produces keys for RAGFlow's service_conf.yaml OIDC configuration.
// RAGFlow uses a YAML config file (not env vars) for OAuth.
//
// The "service_conf_yaml" key contains the complete OIDC block ready to be
// patched into the ragflow-service-config ConfigMap's "local.service_conf.yaml" key.
//
// The OIDCClient CR must provide "redirect_uri" via SecretOverrides since the
// operator doesn't know the RAGFlow external hostname.
func ragflow(data OIDCData, overrides map[string]string) map[string]string {
	redirectURI := overrides["redirect_uri"]

	serviceConfYAML := "oauth:\n" +
		"  oidc:\n" +
		"    type: oidc\n" +
		"    icon: sso\n" +
		"    display_name: \"Login with Authentik\"\n" +
		"    client_id: \"" + data.ClientID + "\"\n" +
		"    client_secret: \"" + data.ClientSecret + "\"\n" +
		"    issuer: \"" + data.IssuerURL + "\"\n" +
		"    scope: \"" + data.Scopes + "\"\n" +
		"    redirect_uri: \"" + redirectURI + "\"\n"

	return map[string]string{
		"client_id":         data.ClientID,
		"client_secret":     data.ClientSecret,
		"issuer":            data.IssuerURL,
		"scope":             data.Scopes,
		"service_conf_yaml": serviceConfYAML,
	}
}

// harbor produces keys for Harbor's OIDC configuration.
// The PostSync Job reads these from the harbor-oidc secret and configures
// Harbor via its API (/api/v2.0/configurations).
func harbor(data OIDCData) map[string]string {
	return map[string]string{
		"client_id":     data.ClientID,
		"client_secret": data.ClientSecret,
		"issuer":        data.IssuerURL,
		"scopes":        data.Scopes,
		"authorize_url": data.AuthorizeURL,
		"token_url":     data.TokenURL,
		"userinfo_url":  data.UserinfoURL,
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
