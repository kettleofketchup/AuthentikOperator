---
title: Grafana Profile
---

# Grafana Profile

The `grafana` profile maps OIDC source data to Grafana's [Generic OAuth](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/generic-oauth/) environment variables. This lets Grafana authenticate users against Authentik with no manual secret wiring.

## Key Mapping

| Secret Key | Source Value |
|:-----------|:------------|
| `GF_AUTH_GENERIC_OAUTH_ENABLED` | `"true"` (static) |
| `GF_AUTH_GENERIC_OAUTH_NAME` | `"authentik"` (static) |
| `GF_AUTH_GENERIC_OAUTH_CLIENT_ID` | `clientId` |
| `GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET` | `clientSecret` |
| `GF_AUTH_GENERIC_OAUTH_AUTH_URL` | `authorizeUrl` |
| `GF_AUTH_GENERIC_OAUTH_TOKEN_URL` | `tokenUrl` |
| `GF_AUTH_GENERIC_OAUTH_API_URL` | `userinfoUrl` |
| `GF_AUTH_GENERIC_OAUTH_SCOPES` | `scopes` (default: `openid email profile`) |
| `GF_AUTH_SIGNOUT_REDIRECT_URL` | `logoutUrl` — Authentik end-session endpoint |
| `GF_AUTH_OAUTH_AUTO_LOGIN` | `"true"` (static) — skip Grafana login form |
| `GF_AUTH_GENERIC_OAUTH_ALLOW_ASSIGN_GRAFANA_ADMIN` | `"true"` (static) — Admin role grants Server Admin |

## Example CR

{%include-markdown "../includes/sample-grafana-cr.md"%}

## Common Overrides

### Role Mapping

Map Authentik groups to Grafana roles using a JMESPath expression:

```yaml title="oidcclient.yaml"
spec:
  secretOverrides:
    GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH: >-
      contains(groups, 'admins') && 'Admin' || 'Viewer'
```

### Allow Sign Up

Automatically create Grafana user accounts for authenticated Authentik users:

```yaml title="oidcclient.yaml"
spec:
  secretOverrides:
    GF_AUTH_GENERIC_OAUTH_ALLOW_SIGN_UP: "true"
```

### Custom Scopes

Request additional scopes such as `groups` for role mapping:

```yaml title="oidcclient.yaml"
spec:
  secretOverrides:
    GF_AUTH_GENERIC_OAUTH_SCOPES: "openid email profile groups"
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
        GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET:
          valueFrom:
            secretKeyRef:
              name: grafana-oauth
              key: GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET
    ```

!!! tip "Rollout Restart"
    Enable `rolloutRestart` in the CR to automatically restart Grafana when the secret changes. This ensures Grafana picks up rotated credentials without manual intervention.

## Authentik Blueprint

Create the Grafana OIDC provider in Authentik using a blueprint:

```yaml title="grafana-blueprint.yaml"
version: 1
metadata:
  name: Grafana OIDC Provider
entries:
  - model: authentik_providers_oauth2.oauth2provider
    id: provider-grafana
    attrs:
      name: grafana-oidc
      authorization_flow: !Find [authentik_flows.flow, [slug, default-provider-authorization-flow]]
      client_type: confidential
      redirect_uris: "https://grafana.example.com/login/generic_oauth"
      signing_key: !Find [authentik_crypto.certificatekeypair, [name, authentik Self-signed Certificate]]
      property_mappings:
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, openid]]
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, email]]
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, profile]]

  - model: authentik_core.application
    attrs:
      name: Grafana
      slug: grafana
      provider: !KeyOf provider-grafana
      meta_launch_url: "https://grafana.example.com"
```

!!! info "Blueprint Deployment"
    Authentik auto-generates `client_id` and `client_secret` when the blueprint creates the OAuth2 provider. The operator reads those values via the API -- you never need to copy them manually.
