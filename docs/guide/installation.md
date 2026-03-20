# Installation

This guide covers installing, upgrading, and uninstalling the AuthentikOperator Helm chart.

## Prerequisites

{%include-markdown "../includes/prerequisites-list.md"%}

---

## Helm OCI Install

The chart is published as an OCI artifact on GHCR. Install it directly with `helm install`:

{%include-markdown "../includes/helm-install-cmd.md"%}

| Flag | Purpose |
|------|---------|
| `oci://ghcr.io/kettleofketchup/authentik-operator` | OCI chart reference on GHCR |
| `--version 0.1.2` | Pin to a specific chart version (recommended) |
| `--set authentik.url=...` | **Required.** Base URL of your Authentik instance |
| `--namespace authentik-operator` | Target namespace for the operator |
| `--create-namespace` | Create the namespace if it does not exist |

!!! note "OCI registry login"
    The chart repository is public. No `helm registry login` is required.

---

## Custom Values File

For more complex configurations, create a `values.yaml` and pass it with `-f`:

```yaml title="values.yaml"
authentik:
  url: https://auth.example.com
  bootstrapSecretRef: authentik-bootstrap
  bootstrapSecretKey: bootstrap_token

tokenSecretName: authentik-operator-token
reconcileInterval: 5m

bootstrap:
  enabled: true

resources:
  limits:
    cpu: 200m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 64Mi
```

Then install with:

```bash
helm install authentik-operator \
  oci://ghcr.io/kettleofketchup/authentik-operator \
  --version 0.1.2 \
  -f values.yaml \
  --namespace authentik-operator \
  --create-namespace
```

See [Configuration](configuration.md) for the full reference of available values.

---

## ArgoCD Application

If you manage your cluster with ArgoCD, declare the operator as an `Application`:

```yaml title="argocd-application.yaml"
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: authentik-operator
  namespace: argocd
spec:
  project: default
  source:
    chart: authentik-operator
    repoURL: ghcr.io/kettleofketchup
    targetRevision: 0.1.2
    helm:
      valuesObject:
        authentik:
          url: https://auth.example.com
  destination:
    server: https://kubernetes.default.svc
    namespace: authentik-operator
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

!!! note "Bootstrap Job and ArgoCD"
    The bootstrap Job is annotated as an ArgoCD **PostSync** hook. ArgoCD will run it
    automatically after a successful sync and clean it up on success. See the
    [Bootstrap](bootstrap.md) guide for details.

---

## Verifying the Installation

After installation, confirm the operator is running and the bootstrap completed successfully.

### Check the operator pod

```bash
kubectl get pods -n authentik-operator
```

Expected output:

```text
NAME                                  READY   STATUS    RESTARTS   AGE
authentik-operator-6d8f9b7c4f-x2kpv   1/1     Running   0          45s
```

### Check the bootstrap Job

```bash
kubectl get jobs -n authentik-operator
```

If bootstrap is enabled and ArgoCD is managing the release, the Job will appear briefly and then
be cleaned up. For non-ArgoCD installs, it remains for `ttlSecondsAfterFinished` (default 300s):

```text
NAME                              COMPLETIONS   DURATION   AGE
authentik-operator-bootstrap      1/1           8s         30s
```

### Verify the API token secret

The bootstrap Job creates this secret. Its presence confirms bootstrap succeeded:

```bash
kubectl get secret authentik-operator-token -n authentik-operator
```

### Check OIDCClient CRD

```bash
kubectl get crd oidcclients.auth.kettleofketchup
```

---

## Upgrading

Upgrade with `helm upgrade`, specifying the new chart version:

```bash
helm upgrade authentik-operator \
  oci://ghcr.io/kettleofketchup/authentik-operator \
  --version 0.2.0 \
  -f values.yaml \
  --namespace authentik-operator
```

!!! warning "CRD upgrades require manual application"
    Helm does not upgrade CRDs that were installed with the chart. If a new version ships
    updated CRDs, apply them manually **before** running `helm upgrade`:

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/kettleofketchup/AuthentikOperator/main/chart/crds/auth.kettleofketchup_oidcclients.yaml
    ```

---

## Uninstalling

### Remove the Helm release

```bash
helm uninstall authentik-operator --namespace authentik-operator
```

### Clean up CRDs

Helm does not delete CRDs on uninstall. Remove them manually if you are fully decommissioning
the operator:

```bash
kubectl delete crd oidcclients.auth.kettleofketchup
```

!!! danger "Deleting the CRD removes all OIDCClient resources"
    Deleting the CRD will cascade-delete **every** `OIDCClient` CR in the cluster. The
    Secrets those CRs created are **not** deleted automatically (they are not owned via
    `ownerReferences` since they live in different namespaces).

### Clean up the namespace

```bash
kubectl delete namespace authentik-operator
```

{%include-markdown "../includes/abbreviations.md"%}
