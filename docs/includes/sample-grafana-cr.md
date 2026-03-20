```yaml
apiVersion: auth.kettleofketchup/v1alpha1
kind: OIDCClient
metadata:
  name: grafana-oidc
spec:
  authentik:
    applicationSlug: grafana
  target:
    namespace: monitoring
    secretName: grafana-oauth
  secretProfile: grafana
  secretOverrides:
    GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH: >-
      contains(groups, 'admins') && 'Admin' || 'Viewer'
    GF_AUTH_GENERIC_OAUTH_ALLOW_SIGN_UP: "true"
  rolloutRestart:
    enabled: true
    targetRef:
      kind: Deployment
      name: kube-prometheus-stack-grafana
      namespace: monitoring
```
