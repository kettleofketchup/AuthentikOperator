# Creating OIDCClient Resources

`OIDCClient` is the Custom Resource that tells the operator which Authentik application to read
OIDC credentials from and where to write them as a Kubernetes Secret.

---

## Where to Place OIDCClient CRs

OIDCClient CRs belong in the **consumer application's** Helm chart, not in the operator chart.
This keeps OIDC configuration close to the application that uses it:

```text
services/apps/kube-prometheus-stack/templates/oidcclient.yaml   # Grafana
services/apps/open-webui/templates/oidcclient.yaml              # OpenWebUI
k3s-core/modules/argocd/chart/templates/oidcclient.yaml         # ArgoCD
```

The operator chart only contains the operator itself, RBAC, bootstrap Job, and CRD definitions.
OIDCClient CRs can be created before the operator is running -- the operator will reconcile
them once it starts.

---

## CRD Spec Reference

Every `OIDCClient` CR follows this structure:

```yaml
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: <unique-name>          # Cluster-scoped unique name for this binding
spec:
  authentik:
    applicationSlug: <string>  # (Required) Slug of the Authentik application
  target:
    namespace: <string>        # (Required) Namespace for the generated Secret
    secretName: <string>       # (Required) Name of the generated Secret
  secretProfile: <string>      # Profile for secret key mapping (default: generic)
  secretOverrides:             # (Optional) Extra or overridden key-value pairs
    KEY_NAME: "value"
  rolloutRestart:              # (Optional) Restart a workload on secret changes
    enabled: true
    targetRef:
      kind: Deployment|StatefulSet
      name: <workload-name>
      namespace: <workload-namespace>
```

### Field Details

#### `spec.authentik.applicationSlug`

The slug of the Authentik application to read OIDC credentials from. This must match the slug
defined in your Authentik blueprint or application configuration.

The operator performs a two-step API lookup:

1. `GET /api/v3/core/applications/{slug}/` to find the application
2. `GET /api/v3/providers/oauth2/?application={pk}` to retrieve `client_id` and `client_secret`

#### `spec.target`

Defines where the operator writes the OIDC credentials Secret:

- **`namespace`** -- The namespace where the Secret is created. Does not need to be the operator
  namespace.
- **`secretName`** -- The name of the Secret resource.

The operator creates the Secret if it does not exist, or updates it if the content has changed.

#### `spec.secretProfile`

Selects a built-in key mapping profile that determines how OIDC data is mapped to Secret keys.
Valid values:

| Profile | Use Case |
|---------|----------|
| `grafana` | Grafana `GF_AUTH_GENERIC_OAUTH_*` environment variables |
| `openwebui` | OpenWebUI `OAUTH_*` / `OPENID_*` environment variables |
| `argocd` | ArgoCD `argocd-secret` keys for Dex integration |
| `generic` | Plain keys: `clientId`, `clientSecret`, `authorizeUrl`, etc. |

Default: `generic`

#### `spec.secretOverrides`

A string map that merges on top of the profile output. Use this to:

- Add application-specific settings (e.g. Grafana role mapping)
- Override default values (e.g. custom scopes)
- Inject extra static values the consuming app needs

Overrides take precedence over profile-generated keys with the same name.

#### `spec.rolloutRestart`

When enabled, the operator patches the target workload's pod template with an annotation
containing the secret hash whenever the Secret content changes. This triggers a rolling restart
so the application picks up the new credentials.

- **`enabled`** -- Set to `true` to activate (default: `false`)
- **`targetRef.kind`** -- `Deployment` or `StatefulSet`
- **`targetRef.name`** -- Name of the workload
- **`targetRef.namespace`** -- Namespace of the workload

---

## Source Data

The operator derives these values from the Authentik API and the configured `authentik.url`:

{%include-markdown "../includes/oidc-source-data.md"%}

Profiles map these source values to application-specific Secret keys.

---

## Profile Examples

### Grafana

{%include-markdown "../includes/sample-grafana-cr.md"%}

The `grafana` profile generates keys matching Grafana's `GF_AUTH_GENERIC_OAUTH_*` environment
variables. The example above adds `secretOverrides` for role mapping and sign-up configuration
that are specific to each deployment.

**Generated Secret keys:**

| Secret Key | Source |
|------------|--------|
| `GF_AUTH_GENERIC_OAUTH_ENABLED` | `"true"` (static) |
| `GF_AUTH_GENERIC_OAUTH_NAME` | `"authentik"` (static) |
| `GF_AUTH_GENERIC_OAUTH_CLIENT_ID` | `clientId` |
| `GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET` | `clientSecret` |
| `GF_AUTH_GENERIC_OAUTH_AUTH_URL` | `authorizeUrl` |
| `GF_AUTH_GENERIC_OAUTH_TOKEN_URL` | `tokenUrl` |
| `GF_AUTH_GENERIC_OAUTH_API_URL` | `userinfoUrl` |
| `GF_AUTH_GENERIC_OAUTH_SCOPES` | `"openid email profile"` |

### OpenWebUI

{%include-markdown "../includes/sample-openwebui-cr.md"%}

The `openwebui` profile generates keys matching OpenWebUI's OAuth environment variables.

**Generated Secret keys:**

| Secret Key | Source |
|------------|--------|
| `ENABLE_OAUTH_SIGNUP` | `"true"` (static) |
| `OAUTH_PROVIDER_NAME` | `"authentik"` (static) |
| `OAUTH_CLIENT_ID` | `clientId` |
| `OAUTH_CLIENT_SECRET` | `clientSecret` |
| `OPENID_PROVIDER_URL` | `issuerUrl` |
| `OAUTH_SCOPES` | `"openid email profile"` |

### ArgoCD

{%include-markdown "../includes/sample-argocd-cr.md"%}

The `argocd` profile writes keys into the `argocd-secret` Secret for Dex OIDC integration.
The `dex.config` section in `argocd-cm` must be configured separately through ArgoCD Helm
values to reference these keys.

**Generated Secret keys:**

| Secret Key | Source |
|------------|--------|
| `dex.authentik.clientSecret` | `clientSecret` (referenced as `$dex.authentik.clientSecret` in Dex config) |
| `clientId` | `clientId` (convenience key) |
| `issuerUrl` | `issuerUrl` (convenience key) |

### Generic

{%include-markdown "../includes/sample-generic-cr.md"%}

The `generic` profile uses plain source variable names as Secret keys, suitable for any
application that accepts standard OIDC configuration:

**Generated Secret keys:**

| Secret Key | Source |
|------------|--------|
| `clientId` | `clientId` |
| `clientSecret` | `clientSecret` |
| `authorizeUrl` | `authorizeUrl` |
| `tokenUrl` | `tokenUrl` |
| `userinfoUrl` | `userinfoUrl` |
| `issuerUrl` | `issuerUrl` |
| `scopes` | `"openid email profile"` |

---

## Secret Overrides

Use `secretOverrides` to add keys that the profile does not include, or to override
profile defaults.

### Adding extra keys

```yaml title="Extra Grafana settings"
spec:
  secretProfile: grafana
  secretOverrides:
    GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH: >-
      contains(groups, 'admins') && 'Admin' || 'Viewer'
    GF_AUTH_GENERIC_OAUTH_ALLOW_SIGN_UP: "true"
    GF_AUTH_GENERIC_OAUTH_AUTO_LOGIN: "true"
```

### Overriding profile defaults

```yaml title="Custom scopes"
spec:
  secretProfile: generic
  secretOverrides:
    scopes: "openid email profile groups"
```

The override replaces the profile's default `scopes` value.

### Static-only secrets

Combine the `generic` profile with overrides to build a fully custom Secret:

```yaml title="Custom key names"
spec:
  secretProfile: generic
  secretOverrides:
    CUSTOM_OAUTH_ENABLED: "true"
    CUSTOM_DISPLAY_NAME: "Corporate SSO"
```

!!! note "Override values must be strings"
    All values in `secretOverrides` are strings. Booleans and numbers must be quoted:
    `"true"`, `"3600"`.

---

## Rollout Restart

When the Secret content changes (detected via SHA256 hash comparison), the operator can
trigger a rolling restart of a consuming workload.

```yaml title="Rollout restart configuration"
spec:
  rolloutRestart:
    enabled: true
    targetRef:
      kind: Deployment        # or StatefulSet
      name: my-app
      namespace: my-namespace
```

The operator patches a pod template annotation on the target workload:

```yaml
auth.kettleofketchup/secret-hash: "sha256:abc123..."
```

This changes the pod template spec, causing Kubernetes to perform a rolling update.

!!! note "Cross-namespace restarts"
    The target workload does not need to be in the same namespace as the OIDCClient CR.
    The operator has cluster-wide RBAC for Deployments and StatefulSets.

---

## Checking Status

The operator sets status conditions on each `OIDCClient` CR. Check them with:

```bash
kubectl get oidc
```

Example output:

```text
NAME            SLUG       PROFILE   TARGET NS    READY   SYNCED   AGE
grafana-oidc    grafana    grafana   monitoring   True    True     5m
argocd-oidc     argocd     argocd    argocd       True    True     5m
openwebui-oidc  open-webui openwebui open-webui   True    True     3m
```

For detailed status, use `kubectl describe`:

```bash
kubectl describe oidc grafana-oidc
```

### Status Conditions

| Condition | Meaning |
|-----------|---------|
| `AuthentikProviderFound` | The operator successfully found the OIDC provider in Authentik |
| `SecretSynced` | The target Secret has been created/updated with current credentials |
| `RolloutTriggered` | A rollout restart was triggered (only present when rollout is enabled) |

A condition with `status: "False"` indicates a problem. Check the condition's `message` field
for details:

```bash
kubectl get oidc grafana-oidc -o jsonpath='{.status.conditions}' | jq .
```

---

## Common Patterns

### Deploy OIDCClient with a Helm chart

Add an `oidcclient.yaml` template to your application's Helm chart:

```yaml title="templates/oidcclient.yaml"
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: {{ include "myapp.fullname" . }}-oidc
spec:
  authentik:
    applicationSlug: {{ .Values.oidc.slug }}
  target:
    namespace: {{ .Release.Namespace }}
    secretName: {{ include "myapp.fullname" . }}-oidc-credentials
  secretProfile: generic
  rolloutRestart:
    enabled: true
    targetRef:
      kind: Deployment
      name: {{ include "myapp.fullname" . }}
      namespace: {{ .Release.Namespace }}
```

### Conditional OIDC

Gate the OIDCClient on a Helm value so it is only created when OIDC is enabled:

```yaml title="templates/oidcclient.yaml"
{{- if .Values.oidc.enabled }}
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: {{ include "myapp.fullname" . }}-oidc
spec:
  authentik:
    applicationSlug: {{ .Values.oidc.slug }}
  target:
    namespace: {{ .Release.Namespace }}
    secretName: {{ .Values.oidc.secretName }}
  secretProfile: generic
{{- end }}
```

### Multiple profiles for the same application

If an application needs credentials in two different formats, create two `OIDCClient` CRs
pointing to the same Authentik slug but different profiles:

```yaml title="Generic + custom secrets for the same app"
---
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: myapp-oidc-generic
spec:
  authentik:
    applicationSlug: my-application
  target:
    namespace: my-app
    secretName: myapp-oidc-generic
  secretProfile: generic
---
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: myapp-oidc-custom
spec:
  authentik:
    applicationSlug: my-application
  target:
    namespace: my-app
    secretName: myapp-oidc-custom
  secretProfile: generic
  secretOverrides:
    OAUTH_ENDPOINT: "https://auth.example.com/application/o/my-application/"
```

{%include-markdown "../includes/abbreviations.md"%}
