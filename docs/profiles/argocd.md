---
title: ArgoCD Profile
---

# ArgoCD Profile

The `argocd` profile produces keys for the `argocd-secret` Kubernetes Secret, enabling ArgoCD's [Dex](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/#dex) OIDC connector to authenticate users against Authentik.

!!! warning "Two-Part Configuration Required"
    The ArgoCD OIDC integration requires configuration in **two places**:

    1. **`argocd-secret`** -- managed by this operator via the `OIDCClient` CR
    2. **`argocd-cm` ConfigMap** (`dex.config` key) -- managed separately via ArgoCD Helm values

    The operator **only** manages the secret. You must configure the Dex connector in your ArgoCD Helm values yourself.

## Key Mapping

| Secret Key | Source Value | Purpose |
|:-----------|:------------|:--------|
| `dex.authentik.clientSecret` | `clientSecret` | Referenced by Dex config as `$dex.authentik.clientSecret` |
| `clientId` | `clientId` | Convenience key for use in Dex config |
| `issuerUrl` | `issuerUrl` | Convenience key for use in Dex config |

!!! info "Why So Few Keys?"
    Unlike other profiles, the ArgoCD profile produces only three keys. ArgoCD's Dex connector reads its full configuration from the `dex.config` key in the `argocd-cm` ConfigMap. The secret only needs to hold the `clientSecret` for Dex's variable substitution (`$dex.authentik.clientSecret`), plus convenience keys for reference.

## Example CR

{%include-markdown "../includes/sample-argocd-cr.md"%}

## Step-by-Step Setup

### Step 1: Create the OIDCClient CR

Deploy the CR shown above. The operator will populate `argocd-secret` with the three keys from the mapping table. This CR is typically placed in your ArgoCD Helm chart templates.

!!! tip "Target the Existing Secret"
    The CR targets `argocd-secret` in the `argocd` namespace. This is the same secret ArgoCD already uses for other configuration. The operator merges its keys into this secret without disturbing existing keys.

### Step 2: Configure ArgoCD Helm Values

Add the Dex OIDC connector configuration to your ArgoCD Helm values. This goes into the `argocd-cm` ConfigMap via the `server.config` section:

```yaml title="argocd-values.yaml"
server:
  config:
    dex.config: |
      connectors:
        - type: oidc
          id: authentik
          name: Authentik
          config:
            issuer: https://auth.example.com/application/o/argocd/
            clientID: <your-client-id>
            clientSecret: $dex.authentik.clientSecret
            insecureEnableGroups: true
            scopes:
              - openid
              - profile
              - email
```

Key points about this configuration:

- **`clientSecret: $dex.authentik.clientSecret`** -- Dex substitutes this variable at runtime by looking up the key `dex.authentik.clientSecret` in `argocd-secret`. This is the value the operator writes.
- **`issuer`** -- Set this to your Authentik instance's issuer URL for the ArgoCD application (`{authentikURL}/application/o/argocd/`).
- **`clientID`** -- The client ID from Authentik. You can find this in the Authentik admin UI or retrieve it from the `clientId` key the operator writes to `argocd-secret`.

!!! warning "Static Values in Dex Config"
    The `issuer` and `clientID` values in `dex.config` are **static strings** in the ConfigMap, not variable references. While the operator writes `clientId` and `issuerUrl` to the secret for convenience, Dex only supports `$secret-key` substitution for the `clientSecret` field. You must set the issuer and client ID in your Helm values directly.

### Step 3: Configure RBAC (Optional)

Map Authentik groups to ArgoCD roles in your Helm values:

```yaml title="argocd-values.yaml"
server:
  rbacConfig:
    policy.csv: |
      g, authentik-admins, role:admin
      g, authentik-readonly, role:readonly
    policy.default: role:readonly
    scopes: "[groups, email]"
```

## Authentik Blueprint

Create the ArgoCD OIDC provider in Authentik:

```yaml title="argocd-blueprint.yaml"
version: 1
metadata:
  name: ArgoCD OIDC Provider
entries:
  - model: authentik_providers_oauth2.oauth2provider
    id: provider-argocd
    attrs:
      name: argocd-oidc
      authorization_flow: !Find [authentik_flows.flow, [slug, default-provider-authorization-flow]]
      client_type: confidential
      redirect_uris: "https://argocd.example.com/api/dex/callback"
      signing_key: !Find [authentik_crypto.certificatekeypair, [name, authentik Self-signed Certificate]]
      property_mappings:
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, openid]]
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, email]]
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, profile]]

  - model: authentik_core.application
    attrs:
      name: ArgoCD
      slug: argocd
      provider: !KeyOf provider-argocd
      meta_launch_url: "https://argocd.example.com"
```

!!! info "Redirect URI"
    ArgoCD's Dex callback URL follows the pattern `https://<argocd-host>/api/dex/callback`. Make sure this matches the `redirect_uris` in your Authentik blueprint.
