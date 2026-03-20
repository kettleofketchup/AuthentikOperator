# Prerequisites

## Requirements

{%include-markdown "../includes/prerequisites-list.md"%}

---

## Setting Up the Authentik Bootstrap Token

The operator's bootstrap Job authenticates with Authentik using a pre-shared token to create its own long-lived API token. This requires the Authentik instance to have been started with the `AUTHENTIK_BOOTSTRAP_TOKEN` environment variable set.

### 1. Configure Authentik with a bootstrap token

Add the `AUTHENTIK_BOOTSTRAP_TOKEN` environment variable to your Authentik deployment. If you deploy Authentik with Helm, add it to your values file:

```yaml title="authentik-values.yaml"
authentik:
  secret_key: <your-secret-key>

env:
  AUTHENTIK_BOOTSTRAP_TOKEN:
    value: "<a-long-random-token>"
```

!!! warning
    The bootstrap token grants full admin access to the Authentik API. Use a strong, randomly generated value and treat it as a sensitive credential.

You can generate a suitable token with:

```bash
openssl rand -hex 32
```

### 2. Create the bootstrap secret in Kubernetes

Create a Kubernetes Secret in the namespace where the operator will be deployed, containing the same token value:

```bash
kubectl create namespace authentik-operator

kubectl create secret generic authentik-bootstrap \
  --namespace authentik-operator \
  --from-literal=bootstrap_token="<same-token-as-above>"
```

!!! note
    The secret name (`authentik-bootstrap`) and key (`bootstrap_token`) must match the Helm values `authentik.bootstrapSecretRef` and `authentik.bootstrapSecretKey`. These are the defaults -- only change them if you override those values.

!!! tip "Avoid manual duplication"
    If the bootstrap token already exists in the Authentik namespace, use a **secret reflector** to automatically mirror it to the operator namespace instead of creating it twice. See [Synchronizing the bootstrap token across namespaces](../guide/bootstrap.md#synchronizing-the-bootstrap-token-across-namespaces) for setup with Ember Stack Reflector or Mittwald Kubernetes Replicator.

---

## Creating Authentik OIDC Applications

The operator **reads** OIDC provider details from Authentik but does **not create** them. You must create the OIDC applications and providers in Authentik before the operator can sync their credentials.

The recommended approach is to use **Authentik Blueprints** for declarative configuration.

### Example Blueprint

Below is a blueprint that creates a Grafana OIDC provider and application:

```yaml title="grafana-blueprint.yaml"
version: 1
metadata:
  name: Grafana OIDC Provider
entries:
  - model: authentik_providers_oauth2.oauth2provider
    id: provider-grafana
    attrs:
      name: grafana-oidc
      authorization_flow: !Find [authentik_flows.flow, [slug, default-provider-authorization-flow]]
      client_type: confidential
      redirect_uris: "https://grafana.example.com/login/generic_oauth"
      signing_key: !Find [authentik_crypto.certificatekeypair, [name, authentik Self-signed Certificate]]
      property_mappings:
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, openid]]
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, email]]
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, profile]]

  - model: authentik_core.application
    attrs:
      name: Grafana
      slug: grafana
      provider: !KeyOf provider-grafana
      meta_launch_url: "https://grafana.example.com"
```

!!! tip
    Authentik automatically generates the `client_id` and `client_secret` when creating an OAuth2 provider. The operator then reads these values via the API -- you never need to copy them manually.

### Loading Blueprints

Mount blueprints as a ConfigMap in your Authentik deployment:

```yaml title="authentik-values.yaml"
authentik:
  blueprints:
    configMaps:
      - name: authentik-blueprints
```

Refer to the [Authentik Blueprints documentation](https://docs.goauthentik.io/docs/installation/blueprints) for full details.
