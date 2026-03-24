| Variable | Source |
|----------|--------|
| `clientId` | Authentik API — provider client ID |
| `clientSecret` | Authentik API — provider client secret |
| `authorizeUrl` | `{baseURL}/application/o/authorize/` (global) |
| `tokenUrl` | `{baseURL}/application/o/token/` (global) |
| `userinfoUrl` | `{baseURL}/application/o/userinfo/` (global) |
| `issuerUrl` | `{baseURL}/application/o/{slug}/` (per-app) |
| `logoutUrl` | `{baseURL}/application/o/{slug}/end-session/` (per-app) |
| `scopes` | `openid email profile` (default) |
