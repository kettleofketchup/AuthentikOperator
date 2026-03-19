# Claude Code Instructions for AuthentikOperator

## Project Overview

AuthentikOperator is a Go Kubernetes operator built with Kubebuilder that mirrors OIDC credentials from Authentik into Kubernetes Secrets.

## Architecture

```
.
├── api/v1alpha1/       # CRD type definitions (OIDCClient)
├── cmd/main.go         # Operator entrypoint (controller-runtime manager)
├── internal/
│   ├── controller/     # Reconciler logic
│   ├── authentik/      # Authentik API HTTP client
│   ├── profiles/       # Secret profile mappings (grafana, argocd, etc.)
│   ├── hash/           # Deterministic secret hashing
│   ├── rollout/        # Deployment/StatefulSet restart trigger
│   └── bootstrap/      # Bootstrap Job logic (token creation)
├── config/             # Kubebuilder Kustomize manifests
├── chart/              # Helm chart for deployment
├── just/               # Justfile modules (wrapping Make where needed)
└── docs/               # MkDocs documentation
```

## Key Technologies

- **Language**: Go 1.23+
- **Framework**: Kubebuilder v4 / controller-runtime
- **Testing**: Ginkgo/Gomega (controller tests), standard Go tests (unit tests)
- **CRD Generation**: controller-gen (via `make manifests generate`)

## Build System

Primary build via Kubebuilder's `Makefile`, wrapped by `just` for convenience.

### Quick Reference

```sh
# Build (top-level aliases)
just build              # Build operator binary -> bin/manager
just test               # Run all tests (includes envtest)
just lint               # Format + lint
just generate           # Regenerate CRD manifests and deepcopy

# Make commands (Kubebuilder)
make build              # Build binary
make test               # Run tests with envtest
make manifests          # Generate CRD YAML and RBAC
make generate           # Generate deepcopy methods
make docker-build       # Build Docker image
```

### Documentation

```sh
just docs::serve        # Serve docs locally (localhost:8000)
just docs::build        # Build docs -> public/
```

### Docker

```sh
just docker::build      # Build Docker image
just docker::push       # Push to GHCR
```

## Code Conventions

### Reconciler Pattern

Controllers live in `internal/controller/`. Each reconciler implements `Reconcile(ctx, req) (Result, error)`.

### Error Handling

Return errors up to the reconciler. Use `fmt.Errorf("context: %w", err)` for wrapping. Set status conditions on the CR for user-visible errors.

### RBAC Markers

Add `+kubebuilder:rbac` markers to the reconciler. Run `make manifests` to regenerate.

## Worktree Convention

Use `.worktrees/` for git worktrees:

```sh
git worktree add .worktrees/feature-name -b feature/feature-name
```
