# Quick Start

This guide walks you through installing the operator, creating an OIDCClient custom resource, and verifying that credentials are synced into a Kubernetes Secret.

!!! warning "Before you begin"
    Make sure you have completed all steps in the [Prerequisites](prerequisites.md) page, including creating the bootstrap secret and setting up OIDC applications in Authentik.

---

## Step 1: Install the Operator

Install AuthentikOperator using Helm:

{%include-markdown "../includes/helm-install-cmd.md"%}

Verify the operator pod is running:

```bash
kubectl get pods -n authentik-operator
```

You should see output similar to:

```
NAME                                  READY   STATUS    RESTARTS   AGE
authentik-operator-6d4b8f7c9f-x2k4p   1/1     Running   0          30s
```

---

## Step 2: Verify Bootstrap Completed

The operator runs a bootstrap Job on first install to create its Authentik API token. Verify it completed successfully:

```bash
kubectl get jobs -n authentik-operator
```

Expected output:

```
NAME                            COMPLETIONS   DURATION   AGE
authentik-operator-bootstrap    1/1           8s         45s
```

You can also confirm the API token secret was created:

```bash
kubectl get secret authentik-operator-token -n authentik-operator
```

!!! note
    If the bootstrap Job fails, check its logs for details:

    ```bash
    kubectl logs job/authentik-operator-bootstrap -n authentik-operator
    ```

    Common causes include an incorrect bootstrap token value or the Authentik instance being unreachable.

---

## Step 3: Create Your First OIDCClient

Apply an `OIDCClient` custom resource to sync Grafana's OIDC credentials. This example uses the built-in `grafana` profile and triggers a rollout restart when the secret changes.

{%include-markdown "../includes/sample-grafana-cr.md"%}

Save the above to a file and apply it:

```bash
kubectl apply -f grafana-oidc.yaml
```

!!! tip
    `OIDCClient` CRs are cluster-scoped in terms of what they can target -- the CR itself lives in the operator's namespace (or any namespace), but it can create Secrets in any namespace specified by `spec.target.namespace`.

---

## Step 4: Verify the Secret Was Created

Check that the operator created the Secret in the target namespace:

```bash
kubectl get secret grafana-oauth -n monitoring
```

Inspect the Secret keys to confirm they match the Grafana profile:

```bash
kubectl get secret grafana-oauth -n monitoring -o jsonpath='{.data}' | jq 'keys'
```

Expected keys:

```json
[
  "GF_AUTH_GENERIC_OAUTH_ALLOW_SIGN_UP",
  "GF_AUTH_GENERIC_OAUTH_API_URL",
  "GF_AUTH_GENERIC_OAUTH_AUTH_URL",
  "GF_AUTH_GENERIC_OAUTH_CLIENT_ID",
  "GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET",
  "GF_AUTH_GENERIC_OAUTH_ENABLED",
  "GF_AUTH_GENERIC_OAUTH_NAME",
  "GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH",
  "GF_AUTH_GENERIC_OAUTH_SCOPES",
  "GF_AUTH_GENERIC_OAUTH_TOKEN_URL"
]
```

---

## Step 5: Check CR Status

Inspect the OIDCClient status to confirm everything synced correctly:

```bash
kubectl get oidc grafana-oidc -o yaml
```

Look for the `status` section:

```yaml
status:
  conditions:
    - type: AuthentikProviderFound
      status: "True"
      reason: ProviderFound
      message: "Successfully retrieved provider for application 'grafana'"
    - type: SecretSynced
      status: "True"
      reason: SecretCreated
      message: "Secret monitoring/grafana-oauth created successfully"
    - type: RolloutTriggered
      status: "True"
      reason: RolloutRestarted
      message: "Restarted Deployment monitoring/kube-prometheus-stack-grafana"
  lastSyncTime: "2026-03-20T10:30:00Z"
  secretHash: "sha256:..."
```

You can also use the short-form table view:

```bash
kubectl get oidc
```

```
NAME           SLUG      PROFILE   TARGET NS    READY   SYNCED   AGE
grafana-oidc   grafana   grafana   monitoring   True    True     2m
```

!!! tip
    The `oidc` short name is registered for the `OIDCClient` resource, so you can use `kubectl get oidc` instead of `kubectl get oidcclients`.

---

## What's Next

- Add more `OIDCClient` resources for other applications using the `openwebui`, `argocd`, or `generic` profiles
- Use `secretOverrides` to add application-specific configuration keys
- Enable `rolloutRestart` to automatically restart workloads when credentials change
