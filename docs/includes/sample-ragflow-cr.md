```yaml
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: ragflow-oidc
spec:
  authentik:
    applicationSlug: ragflow
  target:
    namespace: ragflow
    secretName: ragflow-oidc-credentials
  secretProfile: ragflow
```
