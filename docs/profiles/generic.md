---
title: Generic Profile
---

# Generic Profile

The `generic` profile uses plain OIDC variable names as secret keys, without mapping them to any application-specific naming convention. It is the most flexible profile and serves as the default fallback when an unrecognized profile name is specified.

## Key Mapping

| Secret Key | Source Value |
|:-----------|:------------|
| `clientId` | `clientId` |
| `clientSecret` | `clientSecret` |
| `authorizeUrl` | `authorizeUrl` |
| `tokenUrl` | `tokenUrl` |
| `userinfoUrl` | `userinfoUrl` |
| `issuerUrl` | `issuerUrl` |
| `scopes` | `scopes` (default: `openid email profile`) |

!!! info "One-to-One Mapping"
    The generic profile passes through every OIDC source field without renaming. The secret keys match the internal OIDC data field names exactly.

## Example CR

{%include-markdown "../includes/sample-generic-cr.md"%}

## When to Use Generic vs a Named Profile

| Scenario | Recommended Profile |
|:---------|:-------------------|
| Application has a dedicated profile (Grafana, OpenWebUI, ArgoCD) | Use the named profile |
| Application not covered by any built-in profile | Use `generic` |
| You want full control over every key name via overrides | Use `generic` |
| Prototyping or testing OIDC integration | Use `generic` |
| Feeding OIDC data into a custom init container or script | Use `generic` |

!!! tip "Named Profiles Are Just Convenience"
    Named profiles produce the exact same data as `generic` -- they just rename the keys. If you know the environment variable names your application expects, you can always use `generic` with `secretOverrides` to achieve the same result.

## Adapting Generic to a Specific Application

Use `secretOverrides` to rename or add keys on top of the generic base. The overrides merge into the profile output, so you can build a custom mapping for any application.

### Example: Adapting for a Custom Application

Suppose your application reads OIDC config from these environment variables:

- `APP_OAUTH_CLIENT_ID`
- `APP_OAUTH_CLIENT_SECRET`
- `APP_OAUTH_ISSUER`

You can use the generic profile and override with application-specific keys:

```yaml title="oidcclient.yaml"
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: custom-app-oidc
spec:
  authentik:
    applicationSlug: custom-app
  target:
    namespace: custom-app
    secretName: custom-app-oauth
  secretProfile: generic
  secretOverrides:
    APP_OAUTH_CLIENT_ID: "$(clientId)"
    APP_OAUTH_CLIENT_SECRET: "$(clientSecret)"
    APP_OAUTH_ISSUER: "$(issuerUrl)"
```

!!! warning "Override Values Are Static"
    The `secretOverrides` values are **static strings**, not templates. The `$(clientId)` syntax shown above is illustrative -- it does not perform variable substitution. To map generic keys to custom names, use individual `secretKeyRef` references in your Deployment spec instead:

    ```yaml title="deployment.yaml"
    env:
      - name: APP_OAUTH_CLIENT_ID
        valueFrom:
          secretKeyRef:
            name: custom-app-oauth
            key: clientId
      - name: APP_OAUTH_CLIENT_SECRET
        valueFrom:
          secretKeyRef:
            name: custom-app-oauth
            key: clientSecret
      - name: APP_OAUTH_ISSUER
        valueFrom:
          secretKeyRef:
            name: custom-app-oauth
            key: issuerUrl
    ```

This approach keeps the secret in its generic form and lets the Deployment manifest handle the key-to-env-var mapping.
