```bash
helm install authentik-operator \
  oci://ghcr.io/kettleofketchup/authentik-operator \
  --version 0.1.2 \
  --set authentik.url=https://auth.example.com \
  --namespace authentik-operator \
  --create-namespace
```
