---
title: Grafana Profile
---

# Grafana Profile

The `grafana` profile maps OIDC source data to Grafana's [Generic OAuth](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/generic-oauth/) environment variables. This lets Grafana authenticate users against Authentik with no manual secret wiring.

!!! warning "Users Must Have Email Addresses"
    Grafana requires an email address to create user accounts. If your users come from LDAP/Active Directory and the `mail` attribute is not populated, Grafana login will fail with **"Provider didn't return an email address"**. Ensure your LDAP property mappings include an email fallback (e.g., `username@domain`) or that all AD users have the `mail` attribute set.

## Key Mapping

| Secret Key | Source Value |
|:-----------|:------------|
| `GF_AUTH_GENERIC_OAUTH_ENABLED` | `"true"` (static) |
| `GF_AUTH_GENERIC_OAUTH_NAME` | `"authentik"` (static) |
| `GF_AUTH_GENERIC_OAUTH_CLIENT_ID` | `clientId` |
| `GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET` | `clientSecret` |
| `GF_AUTH_GENERIC_OAUTH_AUTH_URL` | `authorizeUrl` |
| `GF_AUTH_GENERIC_OAUTH_TOKEN_URL` | `tokenUrl` |
| `GF_AUTH_GENERIC_OAUTH_API_URL` | `userinfoUrl` (trailing slash stripped) |
| `GF_AUTH_GENERIC_OAUTH_SCOPES` | `scopes` (default: `openid email profile`) |
| `GF_AUTH_SIGNOUT_REDIRECT_URL` | `logoutUrl` — Authentik end-session endpoint |
| `GF_AUTH_OAUTH_AUTO_LOGIN` | `"true"` (static) — skip Grafana login form |
| `GF_AUTH_GENERIC_OAUTH_ALLOW_ASSIGN_GRAFANA_ADMIN` | `"true"` (static) — Admin role grants Server Admin |
| `GF_AUTH_GENERIC_OAUTH_ALLOW_SIGN_UP` | `"true"` (static) — auto-create Grafana accounts |
| `GF_AUTH_GENERIC_OAUTH_EMAIL_ATTRIBUTE_PATH` | `"email"` (static) — extract email from userinfo |
| `GF_AUTH_GENERIC_OAUTH_NAME_ATTRIBUTE_PATH` | `"name"` (static) — extract display name from userinfo |
| `GF_AUTH_GENERIC_OAUTH_LOGIN_ATTRIBUTE_PATH` | `"preferred_username"` (static) — extract login from userinfo |
| `GF_AUTH_GENERIC_OAUTH_USE_PKCE` | `"true"` (static) — PKCE for token exchange |

!!! note "API URL trailing slash"
    The `api_url` has its trailing slash stripped to prevent Grafana 12+ from appending `/emails` to the userinfo endpoint, which Authentik does not serve and returns a 404.

## Example CR

{%include-markdown "../includes/sample-grafana-cr.md"%}

## Common Overrides

### Role Mapping

Map Authentik groups to Grafana roles using a JMESPath expression:

```yaml title="oidcclient.yaml"
spec:
  secretOverrides:
    GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH: >-
      contains(groups, 'Grafana Admins') && 'Admin' || contains(groups, 'Grafana Editors') && 'Editor' || 'Viewer'
    GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_STRICT: "true"
```

### Custom Scopes

Request additional scopes such as `groups` for role mapping:

```yaml title="oidcclient.yaml"
spec:
  secretOverrides:
    GF_AUTH_GENERIC_OAUTH_SCOPES: "openid email profile groups"
```

### TLS CA Certificate

If Authentik uses a private CA, mount the CA cert and reference it:

```yaml title="values.yaml"
grafana:
  extraSecretMounts:
    - name: edge-ca
      secretName: edge-wildcard-tls
      defaultMode: 0440
      mountPath: /etc/ssl/certs/edge-ca.crt
      subPath: tls.crt
      readOnly: true
  grafana.ini:
    auth.generic_oauth:
      tls_client_ca: /etc/ssl/certs/edge-ca.crt
```

## Consuming the Secret in Grafana

The operator creates a Secret with all the `GF_AUTH_GENERIC_OAUTH_*` keys. Grafana reads configuration from environment variables, so you can inject the entire secret using `envFrom`.

=== "envFrom (recommended)"

    Load all keys from the secret as environment variables:

    ```yaml title="values.yaml"
    grafana:
      envFromSecrets:
        - grafana-oauth
    ```

=== "Individual env vars"

    Reference specific keys if you only need a subset:

    ```yaml title="values.yaml"
    grafana:
      env:
        GF_AUTH_GENERIC_OAUTH_CLIENT_ID:
          valueFrom:
            secretKeyRef:
              name: grafana-oauth
              key: GF_AUTH_GENERIC_OAUTH_CLIENT_ID
    ```

!!! tip "Rollout Restart"
    Enable `rolloutRestart` in the CR to automatically restart Grafana when the secret changes. This ensures Grafana picks up rotated credentials without manual intervention.

## Authentik Blueprint

Create the Grafana OIDC provider in Authentik using a blueprint. The `email` scope mapping **must** be included — without it, the userinfo endpoint won't return the user's email and Grafana login will fail.

```yaml title="grafana-blueprint.yaml"
version: 1
metadata:
  name: Grafana OIDC Provider
entries:
  # Custom groups scope for role mapping
  - model: authentik_providers_oauth2.scopemapping
    identifiers:
      name: "OIDC Groups"
    id: groups-scope
    attrs:
      scope_name: groups
      description: "User's group memberships"
      expression: |
        return {"groups": [group.name for group in request.user.groups.all()]}

  - model: authentik_providers_oauth2.oauth2provider
    identifiers:
      name: grafana-oidc
    id: oidc-provider
    attrs:
      authorization_flow: !Find [authentik_flows.flow, [slug, default-provider-authorization-implicit-consent]]
      invalidation_flow: !Find [authentik_flows.flow, [slug, default-provider-invalidation-flow]]
      client_type: confidential
      client_id: grafana
      redirect_uris:
        - matching_mode: strict
          url: "https://grafana.example.com/login/generic_oauth"
      signing_key: !Find [authentik_crypto.certificatekeypair, [name, authentik Self-signed Certificate]]
      property_mappings:
        - !Find [authentik_providers_oauth2.scopemapping, [managed, goauthentik.io/providers/oauth2/scope-openid]]
        - !Find [authentik_providers_oauth2.scopemapping, [managed, goauthentik.io/providers/oauth2/scope-email]]
        - !Find [authentik_providers_oauth2.scopemapping, [managed, goauthentik.io/providers/oauth2/scope-profile]]
        - !Find [authentik_providers_oauth2.scopemapping, [name, "OIDC Groups"]]

  - model: authentik_core.application
    identifiers:
      slug: grafana
    attrs:
      name: Grafana
      provider: !KeyOf oidc-provider

  # Groups for role mapping
  - model: authentik_core.group
    identifiers:
      name: Grafana Admins
    state: created

  - model: authentik_core.group
    identifiers:
      name: Grafana Editors
    state: created
```

!!! info "Blueprint Deployment"
    Authentik auto-generates `client_id` and `client_secret` when the blueprint creates the OAuth2 provider. The operator reads those values via the API — you never need to copy them manually.
