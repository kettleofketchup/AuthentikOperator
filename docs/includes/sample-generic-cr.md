```yaml
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: myapp-oidc
spec:
  authentik:
    applicationSlug: my-application
  target:
    namespace: my-app
    secretName: myapp-oidc-credentials
  secretProfile: generic
```
