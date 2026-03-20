# OIDCClient CRD Reference

The `OIDCClient` custom resource tells the AuthentikOperator which Authentik application to read OIDC credentials from, where to write the resulting Kubernetes Secret, and optionally which workload to restart when credentials change.

---

## Resource Identity

| Field | Value |
|-------|-------|
| **API Group** | `auth.kettleofketchup` |
| **API Version** | `v1alpha1` |
| **Kind** | `OIDCClient` |
| **List Kind** | `OIDCClientList` |
| **Short Name** | `oidc` |
| **Scope** | Namespaced |

```yaml
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
```

---

## Spec

### `authentik`

Identifies the Authentik application to read OIDC credentials from.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `authentik.applicationSlug` | `string` | Yes | Slug of the Authentik application. Must match the slug configured in Authentik (via blueprints or the admin UI). |

**Validation:**

- `applicationSlug` must be at least 1 character (`minLength: 1`)

---

### `target`

Defines the Kubernetes Secret that the operator creates or updates with the OIDC credentials.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `target.namespace` | `string` | Yes | Namespace to create the Secret in. Can be any namespace in the cluster (cross-namespace). |
| `target.secretName` | `string` | Yes | Name of the Secret to create or update. |

!!! info "Cross-Namespace Secrets"
    The operator writes Secrets to any namespace, not just the namespace where the OIDCClient CR lives. Since Kubernetes does not support cross-namespace `ownerReferences`, the operator uses labels for tracking instead. See [Managed Labels](#managed-labels).

---

### `secretProfile`

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `secretProfile` | `string` | No | `generic` | Selects a built-in key mapping profile that determines the Secret key names. |

**Allowed values:** `grafana`, `openwebui`, `argocd`, `generic`

Each profile maps the OIDC source data into application-specific environment variable names. See [Source Data](#source-data) and [Profile Mappings](#profile-mappings) below.

---

### `secretOverrides`

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `secretOverrides` | `map[string]string` | No | `{}` | Arbitrary key-value pairs merged on top of the profile output. Use to add extra static values or override profile defaults. |

Overrides are applied **after** the profile, so they can replace any profile-generated key or add new keys that the profile does not produce.

---

### `rolloutRestart`

Configures automatic workload restarts when the managed Secret changes.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rolloutRestart.enabled` | `bool` | No | `false` | Whether to trigger a rolling restart on Secret change. |
| `rolloutRestart.targetRef.kind` | `string` | No | -- | Kind of the target workload. |
| `rolloutRestart.targetRef.name` | `string` | No | -- | Name of the target workload. |
| `rolloutRestart.targetRef.namespace` | `string` | No | -- | Namespace of the target workload. |

**Validation on `targetRef.kind`:**

- Must be one of: `Deployment`, `StatefulSet`

!!! tip "How Rollout Restart Works"
    The operator patches the pod template annotation `auth.kettleofketchup/secret-hash` on the target Deployment or StatefulSet with the new Secret hash. Kubernetes sees the annotation change and performs a rolling restart of all pods.

---

## Source Data

The operator derives the following OIDC values from the Authentik API and the configured base URL. These values are the inputs to every profile.

{%
  include-markdown "../includes/oidc-source-data.md"
%}

---

## Profile Mappings

### Profile: `grafana`

Produces Grafana Generic OAuth environment variables.

| Secret Key | Source Value |
|------------|-------------|
| `GF_AUTH_GENERIC_OAUTH_ENABLED` | `"true"` (static) |
| `GF_AUTH_GENERIC_OAUTH_NAME` | `"authentik"` (static) |
| `GF_AUTH_GENERIC_OAUTH_CLIENT_ID` | `clientId` |
| `GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET` | `clientSecret` |
| `GF_AUTH_GENERIC_OAUTH_AUTH_URL` | `authorizeUrl` |
| `GF_AUTH_GENERIC_OAUTH_TOKEN_URL` | `tokenUrl` |
| `GF_AUTH_GENERIC_OAUTH_API_URL` | `userinfoUrl` |
| `GF_AUTH_GENERIC_OAUTH_SCOPES` | `scopes` |

### Profile: `openwebui`

Produces Open WebUI OAuth environment variables.

| Secret Key | Source Value |
|------------|-------------|
| `ENABLE_OAUTH_SIGNUP` | `"true"` (static) |
| `OAUTH_PROVIDER_NAME` | `"authentik"` (static) |
| `OAUTH_CLIENT_ID` | `clientId` |
| `OAUTH_CLIENT_SECRET` | `clientSecret` |
| `OPENID_PROVIDER_URL` | `issuerUrl` |
| `OAUTH_SCOPES` | `scopes` |

### Profile: `argocd`

Produces keys for the `argocd-secret` Secret used by ArgoCD's Dex integration.

| Secret Key | Source Value |
|------------|-------------|
| `dex.authentik.clientSecret` | `clientSecret` |
| `clientId` | `clientId` |
| `issuerUrl` | `issuerUrl` |

!!! warning "ArgoCD Dex Configuration"
    The `dex.config` in the `argocd-cm` ConfigMap is **not** managed by this operator. Configure it in your ArgoCD Helm values with `$dex.authentik.clientSecret` referencing the Secret key produced by this profile.

### Profile: `generic`

Passes through all source data values using their original names.

| Secret Key | Source Value |
|------------|-------------|
| `clientId` | `clientId` |
| `clientSecret` | `clientSecret` |
| `authorizeUrl` | `authorizeUrl` |
| `tokenUrl` | `tokenUrl` |
| `userinfoUrl` | `userinfoUrl` |
| `issuerUrl` | `issuerUrl` |
| `scopes` | `scopes` |

---

## Status

The operator writes the following fields to `.status` after each reconciliation.

### `conditions`

Standard Kubernetes conditions (`metav1.Condition`). Listed by type:

| Condition Type | Meaning when `True` | Meaning when `False` |
|----------------|---------------------|----------------------|
| `AuthentikProviderFound` | The Authentik API returned a valid OAuth2 provider for the configured slug. | The provider could not be found or the Authentik API is unreachable. |
| `SecretSynced` | The target Secret has been created or updated successfully. | The Secret could not be created or updated (e.g., namespace does not exist, RBAC error). |
| `RolloutTriggered` | The target workload was patched to trigger a rolling restart. | The rollout patch failed (e.g., workload not found, RBAC error). Only set when `rolloutRestart.enabled: true`. |

Each condition includes `reason`, `message`, and `observedGeneration` for debugging.

### `lastSyncTime`

| Field | Type | Description |
|-------|------|-------------|
| `lastSyncTime` | `metav1.Time` | Timestamp of the last successful sync with Authentik. |

### `secretHash`

| Field | Type | Description |
|-------|------|-------------|
| `secretHash` | `string` | SHA256 hash of the current Secret data (format: `sha256:<hex>`). Used for change detection -- the operator skips writes when the hash matches. |

---

## Print Columns

When you run `kubectl get oidc`, the following columns are displayed:

| Column | Source |
|--------|--------|
| **Slug** | `.spec.authentik.applicationSlug` |
| **Profile** | `.spec.secretProfile` |
| **Target NS** | `.spec.target.namespace` |
| **Ready** | `.status.conditions[?(@.type=="AuthentikProviderFound")].status` |
| **Synced** | `.status.conditions[?(@.type=="SecretSynced")].status` |
| **Age** | `.metadata.creationTimestamp` |

```bash
$ kubectl get oidc
NAME             SLUG       PROFILE    TARGET NS    READY   SYNCED   AGE
grafana-oidc     grafana    grafana    monitoring   True    True     3d
openwebui-oidc   open-webui openwebui  open-webui   True    True     3d
argocd-oidc      argocd     argocd     argocd       True    True     3d
```

---

## Managed Labels

Secrets created by the operator carry the following labels for identification and garbage collection:

```yaml
labels:
  auth.kettleofketchup/managed-by: authentik-operator
  auth.kettleofketchup/oidc-client: <oidcclient-cr-name>
```

---

## Full Example

{%
  include-markdown "../includes/sample-grafana-cr.md"
%}

??? example "OpenWebUI"

    {%
      include-markdown "../includes/sample-openwebui-cr.md"
    %}

??? example "ArgoCD"

    {%
      include-markdown "../includes/sample-argocd-cr.md"
    %}

??? example "Generic"

    {%
      include-markdown "../includes/sample-generic-cr.md"
    %}

*[CRD]: Custom Resource Definition
*[CR]: Custom Resource
*[OIDC]: OpenID Connect
*[RBAC]: Role-Based Access Control
