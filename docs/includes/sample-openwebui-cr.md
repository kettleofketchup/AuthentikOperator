```yaml
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: openwebui-oidc
spec:
  authentik:
    applicationSlug: open-webui
  target:
    namespace: open-webui
    secretName: openwebui-oauth
  secretProfile: openwebui
  rolloutRestart:
    enabled: true
    targetRef:
      kind: Deployment
      name: open-webui
      namespace: open-webui
```
