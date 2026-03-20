# Helm Values Reference

Complete reference for all configurable values in the AuthentikOperator Helm chart.

---

## Image

Settings for the operator container image.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `image.repository` | `string` | `ghcr.io/kettleofketchup/authentik-operator` | Container image repository. |
| `image.tag` | `string` | `""` (uses `appVersion`) | Image tag override. When empty, defaults to the chart's `appVersion`. Pin to a specific version tag for production. |
| `image.pullPolicy` | `string` | `IfNotPresent` | Image pull policy. Set to `Always` if using `latest` tag. |
| `imagePullSecrets` | `list` | `[]` | List of image pull secrets for private registries. |

---

## Name Overrides

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `nameOverride` | `string` | `""` | Override the chart name used in resource names. |
| `fullnameOverride` | `string` | `""` | Override the full resource name (release + chart). |

---

## Authentik Connection

Configuration for connecting to the Authentik instance.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `authentik.url` | `string` | `""` | **Required.** Base URL of the Authentik instance (e.g., `https://auth.example.com`). |
| `authentik.bootstrapSecretRef` | `string` | `authentik-bootstrap` | Name of the Kubernetes Secret containing the Authentik bootstrap token. Must exist in the same namespace as the operator. |
| `authentik.bootstrapSecretKey` | `string` | `bootstrap_token` | Key within the bootstrap secret that holds the token value. |

!!! warning "Required Value"
    `authentik.url` is required and the chart will fail to render without it. Set it via `--set authentik.url=https://auth.example.com` or in your values file.

!!! info "Bootstrap Token"
    The bootstrap secret must contain the same token value that was set as the `AUTHENTIK_BOOTSTRAP_TOKEN` environment variable on the Authentik instance. This token is used only once during the bootstrap Job to create a long-lived API token.

---

## Bootstrap

Settings for the one-time bootstrap Job that creates the operator's Authentik API token.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `bootstrap.enabled` | `bool` | `true` | Whether to create the bootstrap Job. Disable if you pre-provision the API token secret manually. |
| `bootstrap.ttlSecondsAfterFinished` | `int` | `300` | Seconds after Job completion before Kubernetes cleans it up (for non-ArgoCD users). ArgoCD users rely on `HookSucceeded` delete policy instead. |
| `bootstrap.activeDeadlineSeconds` | `int` | `600` | Upper bound on total bootstrap retry duration. The Job fails if it cannot complete within this window. |

!!! tip "ArgoCD Integration"
    The bootstrap Job is annotated with `argocd.argoproj.io/hook: PostSync` and `argocd.argoproj.io/hook-delete-policy: HookSucceeded`. ArgoCD runs it after a successful sync and deletes it on success.

---

## Operator Settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tokenSecretName` | `string` | `authentik-operator-token` | Name of the Kubernetes Secret where the bootstrap Job writes the Authentik API token. The operator Deployment reads from this secret at startup. |
| `reconcileInterval` | `string` | `5m` | How often the operator re-syncs each OIDCClient CR with Authentik. Accepts Go duration strings (`30s`, `5m`, `1h`). |
| `leaderElect` | `bool` | `true` | Enable leader election for HA deployments. Ensures only one active controller manager when running multiple replicas. |
| `healthProbePort` | `int` | `8081` | Port for liveness (`/healthz`) and readiness (`/readyz`) probes. |

---

## Service Account

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `serviceAccount.create` | `bool` | `true` | Whether to create a ServiceAccount. Set to `false` to use an existing one. |
| `serviceAccount.name` | `string` | `""` | ServiceAccount name. Leave empty to auto-derive from the release fullname. |
| `serviceAccount.annotations` | `map` | `{}` | Annotations to add to the ServiceAccount (e.g., for IAM role bindings). |

---

## Resources and Scheduling

### Resource Limits

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `resources.limits.cpu` | `string` | `200m` | CPU limit for the operator container. |
| `resources.limits.memory` | `string` | `128Mi` | Memory limit for the operator container. |
| `resources.requests.cpu` | `string` | `100m` | CPU request for the operator container. |
| `resources.requests.memory` | `string` | `64Mi` | Memory request for the operator container. |

### Scheduling Constraints

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `nodeSelector` | `map` | `{}` | Node selector labels for pod scheduling. |
| `tolerations` | `list` | `[]` | Tolerations for pod scheduling on tainted nodes. |
| `affinity` | `map` | `{}` | Affinity and anti-affinity rules for pod scheduling. |

---

## Pod Customization

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `podAnnotations` | `map` | `{}` | Annotations to add to the operator pod. |
| `podLabels` | `map` | `{}` | Extra labels to add to the operator pod (in addition to selector labels). |

---

## Default `values.yaml`

??? note "Full default values.yaml"

    ```yaml
    image:
      repository: ghcr.io/kettleofketchup/authentik-operator
      # Pin to a specific version tag for production; "latest" requires pullPolicy: Always
      tag: ""
      pullPolicy: IfNotPresent

    imagePullSecrets: []

    nameOverride: ""
    fullnameOverride: ""

    authentik:
      # REQUIRED: Base URL of the Authentik instance
      url: ""
      # Name of the K8s secret containing the bootstrap token (AUTHENTIK_BOOTSTRAP_TOKEN)
      # This secret must exist in the SAME namespace as the operator
      bootstrapSecretRef: authentik-bootstrap
      # Key within the bootstrap secret that holds the token
      bootstrapSecretKey: bootstrap_token

    tokenSecretName: authentik-operator-token

    reconcileInterval: 5m

    bootstrap:
      enabled: true
      # Seconds after completion before K8s cleans up the Job (for non-ArgoCD users)
      ttlSecondsAfterFinished: 300
      # Upper bound on bootstrap retry duration
      activeDeadlineSeconds: 600

    serviceAccount:
      create: true
      # Leave empty to auto-derive from release fullname
      name: ""
      annotations: {}

    leaderElect: true

    healthProbePort: 8081

    resources:
      limits:
        cpu: 200m
        memory: 128Mi
      requests:
        cpu: 100m
        memory: 64Mi

    nodeSelector: {}
    tolerations: []
    affinity: {}
    podAnnotations: {}
    podLabels: {}
    ```

---

## Minimal Install Example

{%
  include-markdown "../includes/helm-install-cmd.md"
%}

*[CRD]: Custom Resource Definition
*[OIDC]: OpenID Connect
*[RBAC]: Role-Based Access Control
*[GHCR]: GitHub Container Registry
