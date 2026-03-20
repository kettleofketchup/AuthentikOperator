---
title: OpenWebUI Profile
---

# OpenWebUI Profile

The `openwebui` profile maps OIDC source data to [OpenWebUI's](https://docs.openwebui.com/) OAuth environment variables. OpenWebUI uses the OpenID Connect discovery URL to auto-configure endpoints, so this profile provides the issuer URL rather than individual endpoint URLs.

## Key Mapping

| Secret Key | Source Value |
|:-----------|:------------|
| `ENABLE_OAUTH_SIGNUP` | `"true"` (static) |
| `OAUTH_PROVIDER_NAME` | `"authentik"` (static) |
| `OAUTH_CLIENT_ID` | `clientId` |
| `OAUTH_CLIENT_SECRET` | `clientSecret` |
| `OPENID_PROVIDER_URL` | `issuerUrl` |
| `OAUTH_SCOPES` | `scopes` (default: `openid email profile`) |

!!! info "OpenID Discovery"
    OpenWebUI uses `OPENID_PROVIDER_URL` to fetch the `.well-known/openid-configuration` document, which provides all necessary endpoint URLs automatically. This is why the profile maps `issuerUrl` instead of individual authorize/token/userinfo URLs.

## Example CR

{%include-markdown "../includes/sample-openwebui-cr.md"%}

## Consuming the Secret in OpenWebUI

The operator creates a Secret containing all the `OAUTH_*` and `OPENID_*` keys. Inject them into the OpenWebUI deployment using `envFrom`.

=== "envFrom (recommended)"

    ```yaml title="values.yaml"
    extraEnvVars:
      - secretRef:
          name: openwebui-oauth
    ```

=== "Helm values env block"

    If your OpenWebUI Helm chart supports an `env` map:

    ```yaml title="values.yaml"
    env:
      - name: OAUTH_CLIENT_ID
        valueFrom:
          secretKeyRef:
            name: openwebui-oauth
            key: OAUTH_CLIENT_ID
      - name: OAUTH_CLIENT_SECRET
        valueFrom:
          secretKeyRef:
            name: openwebui-oauth
            key: OAUTH_CLIENT_SECRET
      - name: OPENID_PROVIDER_URL
        valueFrom:
          secretKeyRef:
            name: openwebui-oauth
            key: OPENID_PROVIDER_URL
      - name: ENABLE_OAUTH_SIGNUP
        valueFrom:
          secretKeyRef:
            name: openwebui-oauth
            key: ENABLE_OAUTH_SIGNUP
      - name: OAUTH_PROVIDER_NAME
        valueFrom:
          secretKeyRef:
            name: openwebui-oauth
            key: OAUTH_PROVIDER_NAME
      - name: OAUTH_SCOPES
        valueFrom:
          secretKeyRef:
            name: openwebui-oauth
            key: OAUTH_SCOPES
    ```

!!! tip "Rollout Restart"
    The example CR enables `rolloutRestart` targeting the `open-webui` Deployment. When the operator detects a credential change, it triggers a rolling restart so OpenWebUI picks up the new values automatically.
