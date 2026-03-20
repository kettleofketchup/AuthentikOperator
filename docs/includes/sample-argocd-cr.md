```yaml
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: argocd-oidc
spec:
  authentik:
    applicationSlug: argocd
  target:
    namespace: argocd
    secretName: argocd-secret
  secretProfile: argocd
```
