---
title: Secret Profiles
---

# Secret Profiles

## What Are Profiles?

Every application that supports OIDC/OAuth2 authentication needs the same core data -- a client ID, a client secret, endpoint URLs, and scopes. However, each application expects that data under **different environment variable names**.

Grafana expects `GF_AUTH_GENERIC_OAUTH_CLIENT_ID`. OpenWebUI expects `OAUTH_CLIENT_ID`. ArgoCD expects the secret value stored under `dex.authentik.clientSecret`. The underlying data is identical; only the key names differ.

**Profiles** solve this mapping problem. A profile is a named transformation that takes OIDC source data from the Authentik API and produces a Kubernetes Secret with the correct key names for a specific application. Instead of manually specifying every key in `secretOverrides`, you set `secretProfile: grafana` and the operator handles the rest.

## OIDC Source Data

The operator retrieves and derives the following values from the Authentik API for every `OIDCClient` CR:

{%include-markdown "../includes/oidc-source-data.md"%}

!!! info "Global vs Per-App Endpoints"
    The `authorizeUrl`, `tokenUrl`, and `userinfoUrl` endpoints are **global** in Authentik -- they do not change per application. Only `issuerUrl` is unique to each application slug.

## Available Profiles

| Profile | Target Application | Description |
|:--------|:-------------------|:------------|
| [`grafana`](grafana.md) | Grafana | Maps to Grafana's Generic OAuth environment variables |
| [`openwebui`](openwebui.md) | OpenWebUI | Maps to OpenWebUI's OAuth environment variables |
| [`argocd`](argocd.md) | ArgoCD | Produces keys for `argocd-secret` for Dex OIDC integration |
| [`generic`](generic.md) | Any application | Uses plain OIDC variable names as secret keys |

!!! tip "Fallback Behavior"
    If an unrecognized profile name is specified, the operator falls back to the `generic` profile.

## Secret Overrides

Every profile produces a base set of key-value pairs. The `secretOverrides` field in the `OIDCClient` CR lets you **add or replace** keys on top of the profile output.

Overrides are merged after the profile runs, so they take precedence over any profile-generated value.

```yaml title="oidcclient.yaml"
spec:
  secretProfile: grafana
  secretOverrides:
    # Override a profile-generated key
    GF_AUTH_GENERIC_OAUTH_SCOPES: "openid email profile groups"
    # Add a key the profile does not produce
    GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH: >-
      contains(groups, 'admins') && 'Admin' || 'Viewer'
```

Common use cases for overrides:

- **Custom scopes** -- add `groups` or other scopes beyond the default `openid email profile`
- **Application-specific settings** -- Grafana role mapping, sign-up policies, etc.
- **Extra static values** -- any additional key-value pairs the consuming application needs
- **Adapting the generic profile** -- use `generic` as a base and override keys to match a specific app

!!! warning "Override Keys Must Be Exact"
    Override keys are used verbatim as Secret data keys. Ensure they match exactly what your application expects, including casing and punctuation.
