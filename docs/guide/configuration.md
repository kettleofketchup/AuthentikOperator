# Configuration

All operator behaviour is controlled through Helm chart values. This page documents every
available option.

---

## Values Reference

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `authentik.url` | string | `""` | **(Required)** Base URL of the Authentik instance (e.g. `https://auth.example.com`) |
| `authentik.bootstrapSecretRef` | string | `authentik-bootstrap` | Name of the K8s Secret containing the Authentik bootstrap token. Must exist in the operator namespace. |
| `authentik.bootstrapSecretKey` | string | `bootstrap_token` | Key within the bootstrap secret that holds the token value |
| `tokenSecretName` | string | `authentik-operator-token` | Name of the K8s Secret where the operator stores its long-lived API token (created by the bootstrap Job) |
| `reconcileInterval` | string | `5m` | How often the operator re-syncs each `OIDCClient` with Authentik |
| `bootstrap.enabled` | bool | `true` | Whether to create the bootstrap Job that provisions the API token |
| `bootstrap.ttlSecondsAfterFinished` | int | `300` | Seconds after Job completion before K8s garbage-collects it |
| `bootstrap.activeDeadlineSeconds` | int | `600` | Maximum duration the bootstrap Job may run before being terminated |
| `serviceAccount.create` | bool | `true` | Whether to create a ServiceAccount for the operator |
| `serviceAccount.name` | string | `""` | Override the ServiceAccount name. Defaults to the Helm release fullname. |
| `serviceAccount.annotations` | object | `{}` | Annotations to add to the ServiceAccount (e.g. for IRSA / Workload Identity) |
| `leaderElect` | bool | `true` | Enable leader election for high-availability deployments |
| `healthProbePort` | int | `8081` | Port for liveness and readiness probes |
| `image.repository` | string | `ghcr.io/kettleofketchup/authentik-operator` | Container image repository |
| `image.tag` | string | `""` | Image tag. Defaults to the chart `appVersion` when empty. |
| `image.pullPolicy` | string | `IfNotPresent` | Kubernetes image pull policy |
| `imagePullSecrets` | list | `[]` | List of image pull secret references |
| `nameOverride` | string | `""` | Override the chart name used in resource names |
| `fullnameOverride` | string | `""` | Override the full release name used in resource names |
| `resources.limits.cpu` | string | `200m` | CPU limit |
| `resources.limits.memory` | string | `128Mi` | Memory limit |
| `resources.requests.cpu` | string | `100m` | CPU request |
| `resources.requests.memory` | string | `64Mi` | Memory request |
| `nodeSelector` | object | `{}` | Node selector for pod scheduling |
| `tolerations` | list | `[]` | Tolerations for pod scheduling |
| `affinity` | object | `{}` | Affinity rules for pod scheduling |
| `podAnnotations` | object | `{}` | Annotations added to the operator pod |
| `podLabels` | object | `{}` | Labels added to the operator pod |

---

## Example Configurations

### Minimal

The only required value is `authentik.url`. Everything else uses sensible defaults:

```yaml title="values-minimal.yaml"
authentik:
  url: https://auth.example.com
```

### Production

A production-ready configuration with explicit resource limits, pinned image tag, and
node affinity:

```yaml title="values-production.yaml"
image:
  tag: "0.1.2"
  pullPolicy: IfNotPresent

authentik:
  url: https://auth.example.com
  bootstrapSecretRef: authentik-bootstrap
  bootstrapSecretKey: bootstrap_token

tokenSecretName: authentik-operator-token
reconcileInterval: 5m

bootstrap:
  enabled: true
  ttlSecondsAfterFinished: 300
  activeDeadlineSeconds: 600

leaderElect: true

resources:
  limits:
    cpu: 200m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 64Mi

nodeSelector:
  kubernetes.io/os: linux

tolerations:
  - key: "node-role.kubernetes.io/control-plane"
    operator: "Exists"
    effect: "NoSchedule"
```

### Custom Namespace with Existing ServiceAccount

When deploying into a shared namespace with a pre-existing ServiceAccount:

```yaml title="values-custom-ns.yaml"
authentik:
  url: https://auth.example.com

serviceAccount:
  create: false
  name: my-existing-sa

bootstrap:
  enabled: false  # Token already provisioned manually
```

!!! note "Disabling bootstrap"
    If you set `bootstrap.enabled: false`, you must create the API token secret manually.
    See [Bootstrap](bootstrap.md) for details on the expected secret format.

---

## Authentik URL

The `authentik.url` value is the base URL of your Authentik instance. It is used to:

- Derive OIDC endpoint URLs (authorize, token, userinfo, issuer)
- Connect to the Authentik API during reconciliation
- Authenticate during bootstrap

```yaml
authentik:
  url: https://auth.example.com  # No trailing slash
```

!!! warning "URL format"
    Do not include a trailing slash or API path. The operator appends paths like
    `/api/v3/core/applications/` automatically.

---

## Reconcile Interval

The `reconcileInterval` controls how frequently the operator re-checks each `OIDCClient` CR
against the Authentik API, even if nothing has changed. This is the fallback cadence; the
operator also reconciles immediately on CR changes.

```yaml
reconcileInterval: 5m  # Default: 5 minutes
```

Valid duration strings follow the Go `time.Duration` format: `30s`, `5m`, `1h`.

---

## Bootstrap Configuration

The `bootstrap` section controls the one-time Job that creates the operator's Authentik API
token. See [Bootstrap](bootstrap.md) for a detailed walkthrough.

```yaml
bootstrap:
  enabled: true                   # Set false if provisioning the token manually
  ttlSecondsAfterFinished: 300    # Cleanup delay (non-ArgoCD)
  activeDeadlineSeconds: 600      # Timeout for bootstrap retries
```

---

## Scheduling

Use standard Kubernetes scheduling fields to control where the operator pod runs:

=== "Node Selector"

    ```yaml title="values.yaml"
    nodeSelector:
      kubernetes.io/os: linux
      topology.kubernetes.io/zone: us-east-1a
    ```

=== "Tolerations"

    ```yaml title="values.yaml"
    tolerations:
      - key: "dedicated"
        operator: "Equal"
        value: "operators"
        effect: "NoSchedule"
    ```

=== "Affinity"

    ```yaml title="values.yaml"
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
            - matchExpressions:
                - key: node-role.kubernetes.io/control-plane
                  operator: Exists
    ```

{%include-markdown "../includes/abbreviations.md"%}
