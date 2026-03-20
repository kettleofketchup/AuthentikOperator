# RagFlow Profile

The `ragflow` profile produces OIDC credential keys for [RagFlow](https://github.com/infiniflow/ragflow), an open-source RAG (Retrieval-Augmented Generation) application.

!!! info "Config file, not environment variables"
    Unlike Grafana or OpenWebUI, RagFlow configures OAuth via its `service_conf.yaml` file, not through environment variables. This profile provides the credential values as Secret keys that you reference when building your RagFlow configuration.

---

## Key Mapping

| Secret Key | Source | Description |
|------------|--------|-------------|
| `client_id` | Authentik provider client ID | OAuth client identifier |
| `client_secret` | Authentik provider client secret | OAuth client secret |
| `issuer` | `{baseURL}/application/o/{slug}/` | OIDC issuer URL (enables auto-discovery) |
| `scope` | `openid email profile` | Requested OAuth scopes |
| `redirect_uri_path` | `/v1/user/oauth/callback/authentik` | Callback path to append to your RagFlow URL |

---

## Example OIDCClient

{%include-markdown "../includes/sample-ragflow-cr.md"%}

---

## Configuring RagFlow

RagFlow reads OAuth settings from the `oauth` section of `service_conf.yaml`. Use the Secret values produced by this profile to populate the configuration.

### Using OIDC (recommended)

Since Authentik supports OIDC discovery, RagFlow only needs the `issuer` URL — it will automatically fetch the authorization, token, and userinfo endpoints:

```yaml title="service_conf.yaml"
oauth:
  authentik:
    type: "oidc"
    display_name: "Authentik SSO"
    icon: "sso"
    client_id: "<from client_id key>"
    client_secret: "<from client_secret key>"
    issuer: "<from issuer key>"
    scope: "<from scope key>"
    redirect_uri: "https://ragflow.example.com/v1/user/oauth/callback/authentik"
```

!!! tip "Redirect URI"
    Combine your RagFlow domain with the `redirect_uri_path` from the Secret:
    `https://your-ragflow-domain` + `/v1/user/oauth/callback/authentik`

### Mounting the Secret values

There are several ways to get the Secret values into RagFlow's config:

=== "Init container with envsubst"

    Mount the Secret as environment variables on an init container that templates `service_conf.yaml`:

    ```yaml title="ragflow-deployment.yaml"
    initContainers:
      - name: config-render
        image: bhgedigital/envsubst
        envFrom:
          - secretRef:
              name: ragflow-oidc-credentials
        command: ["sh", "-c"]
        args:
          - envsubst < /config-template/service_conf.yaml > /config/service_conf.yaml
        volumeMounts:
          - name: config-template
            mountPath: /config-template
          - name: config
            mountPath: /config
    ```

=== "External config management"

    If you use Kustomize or Helm to template RagFlow's config, reference the Secret values
    directly in your templating layer. The operator keeps the Secret in sync — your config
    pipeline reads from it.

---

## Authentik Blueprint

Create the RagFlow OIDC provider in Authentik:

```yaml title="ragflow-blueprint.yaml"
version: 1
metadata:
  name: RagFlow OIDC Provider
entries:
  - model: authentik_providers_oauth2.oauth2provider
    id: provider-ragflow
    attrs:
      name: ragflow-oidc
      authorization_flow: !Find [authentik_flows.flow, [slug, default-provider-authorization-flow]]
      client_type: confidential
      redirect_uris: "https://ragflow.example.com/v1/user/oauth/callback/authentik"
      signing_key: !Find [authentik_crypto.certificatekeypair, [name, authentik Self-signed Certificate]]
      property_mappings:
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, openid]]
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, email]]
        - !Find [authentik_providers_oauth2.scopemapping, [scope_name, profile]]

  - model: authentik_core.application
    attrs:
      name: RagFlow
      slug: ragflow
      provider: !KeyOf provider-ragflow
      meta_launch_url: "https://ragflow.example.com"
```

---

## When to Use This Profile

| Scenario | Profile |
|----------|---------|
| RagFlow with Authentik OIDC | `ragflow` |
| RagFlow with a non-Authentik provider | `generic` (map keys manually via `secretOverrides`) |
| Need additional OAuth fields | `ragflow` + `secretOverrides` for extra keys |
