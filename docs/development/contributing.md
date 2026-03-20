# Development Guide

This guide covers everything you need to build, test, and contribute to AuthentikOperator.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| [Go](https://go.dev/dl/) | 1.25+ | Language runtime |
| [Kubebuilder](https://book.kubebuilder.io/quick-start#installation) | v4 | Operator scaffolding and code generation |
| [just](https://github.com/casey/just#installation) | latest | Task runner (wraps Make targets) |
| [Docker](https://docs.docker.com/get-docker/) | latest | Container image builds |
| [Helm](https://helm.sh/docs/intro/install/) | v3 | Chart packaging and deployment |
| [golangci-lint](https://golangci-lint.run/welcome/install/) | v2.8+ | Go linter (auto-installed by Make) |
| [uv](https://docs.astral.sh/uv/) | latest | Python tool runner for MkDocs |

---

## Getting Started

Clone the repository and bootstrap the development environment:

```bash
git clone https://github.com/kettleofketchup/AuthentikOperator.git
cd AuthentikOperator
```

Run the dev setup script to install Python dependencies (for docs tooling):

```bash
just dev
```

Build the operator binary and run the test suite to verify everything works:

```bash
just build
just test
```

The binary is written to `bin/manager`.

---

## Build Commands

### Just Recipes (Top-Level)

| Command | Description |
|---------|-------------|
| `just build` | Build the operator binary to `bin/manager` |
| `just test` | Run all tests (unit + envtest) |
| `just lint` | Format code and run golangci-lint |
| `just generate` | Regenerate CRD manifests and deepcopy methods |
| `just run` | Build and run the operator locally (outside a cluster) |
| `just dev` | Bootstrap the development environment |

### Just Module Recipes

| Command | Description |
|---------|-------------|
| `just go::build` | Build Go binary (delegates to Make) |
| `just go::test` | Run tests (delegates to Make) |
| `just go::lint` | Format and lint Go code |
| `just go::format` | Format Go code (`go fmt`) |
| `just go::tidy` | Tidy Go modules |
| `just go::clean` | Clean build artifacts |
| `just docs::serve` | Start MkDocs dev server with live reload (localhost:8000) |
| `just docs::build` | Build static documentation to `public/` |
| `just docs::clean` | Remove generated documentation |
| `just docker::build` | Build Docker image |
| `just docker::push` | Build and push Docker image to GHCR |
| `just testing::coverage` | Run tests with coverage report (`cover.out`) |
| `just testing::bench` | Run benchmarks |

### Make Targets (Kubebuilder)

| Command | Description |
|---------|-------------|
| `make build` | Build manager binary (runs manifests, generate, fmt, vet first) |
| `make test` | Run tests with envtest |
| `make manifests` | Generate CRD YAML, RBAC, and webhook configs |
| `make generate` | Generate deepcopy method implementations |
| `make fmt` | Run `go fmt` |
| `make vet` | Run `go vet` |
| `make docker-build` | Build Docker image |
| `make docker-push` | Push Docker image |
| `make install` | Install CRDs into the current cluster |
| `make uninstall` | Remove CRDs from the current cluster |
| `make deploy` | Deploy the controller to the current cluster |
| `make undeploy` | Remove the controller from the current cluster |
| `make lint` | Run golangci-lint |
| `make lint-fix` | Run golangci-lint with auto-fix |
| `make test-e2e` | Run end-to-end tests using Kind |

---

## Project Structure

```
.
├── api/v1alpha1/           # CRD type definitions (OIDCClient)
│   ├── oidcclient_types.go # Spec, Status, and supporting types
│   ├── groupversion_info.go# GroupVersion registration
│   └── zz_generated.deepcopy.go  # Auto-generated deepcopy methods
├── cmd/main.go             # Operator entrypoint (controller-runtime manager)
├── internal/
│   ├── controller/         # Reconciler logic (Reconcile loop)
│   ├── authentik/          # HTTP client for the Authentik REST API
│   ├── profiles/           # Secret profile mappings (grafana, argocd, etc.)
│   ├── hash/               # Deterministic SHA256 hashing for change detection
│   ├── rollout/            # Deployment/StatefulSet restart trigger
│   └── bootstrap/          # Bootstrap Job logic (API token creation)
├── config/                 # Kubebuilder Kustomize manifests (CRDs, RBAC, manager)
├── chart/                  # Helm chart for deployment
├── just/                   # Justfile modules (go, docs, docker, testing, etc.)
├── docs/                   # MkDocs documentation source
├── test/e2e/               # End-to-end tests (Kind-based)
├── Makefile                # Kubebuilder-generated Makefile
├── justfile                # Top-level just task runner config
└── mkdocs.yml              # MkDocs configuration
```

---

## Development Workflow

### Making Code Changes

1. Create a feature branch (or use a worktree):

    ```bash
    git worktree add .worktrees/my-feature -b feature/my-feature
    ```

2. Make your changes in the appropriate package under `internal/` or `api/`.

3. If you modified CRD types in `api/v1alpha1/oidcclient_types.go`, regenerate manifests:

    ```bash
    just generate
    ```

4. Build and test:

    ```bash
    just build
    just test
    ```

5. Lint before committing:

    ```bash
    just lint
    ```

### Running Tests

Tests use [envtest](https://book.kubebuilder.io/reference/envtest) to run a lightweight API server and etcd instance. The `make test` target (called by `just test`) handles downloading the required binaries automatically.

```bash
# Run the full test suite
just test

# Run tests with coverage output
just testing::coverage
# Coverage report is written to cover.out

# Run benchmarks
just testing::bench
```

To run a specific test package:

```bash
go test ./internal/profiles/ -v
```

### Regenerating CRDs

Whenever you modify the types in `api/v1alpha1/` or the RBAC markers (`+kubebuilder:rbac`) in the controller, you must regenerate:

```bash
just generate
# This runs: make manifests generate
```

This regenerates:

- **CRD YAML** in `config/crd/bases/`
- **RBAC ClusterRole** in `config/rbac/`
- **DeepCopy methods** in `api/v1alpha1/zz_generated.deepcopy.go`

!!! warning
    Always commit the regenerated files alongside your type changes. CI will fail if generated files are out of date.

### Linting

The project uses `golangci-lint` for static analysis. The binary is auto-installed to `bin/` on first run.

```bash
# Check for lint issues
just lint

# Auto-fix lint issues where possible
make lint-fix
```

---

## Adding a New Profile

Secret profiles live in `internal/profiles/profiles.go`. Each profile maps OIDC data to application-specific environment variable names.

### Step 1: Add the Profile Function

Add a new function in `internal/profiles/profiles.go` that accepts `OIDCData` and returns a `map[string]string`:

```go
func myapp(data OIDCData) map[string]string {
    return map[string]string{
        "MYAPP_OIDC_CLIENT_ID":     data.ClientID,
        "MYAPP_OIDC_CLIENT_SECRET": data.ClientSecret,
        "MYAPP_OIDC_ISSUER":        data.IssuerURL,
        "MYAPP_OIDC_SCOPES":        data.Scopes,
    }
}
```

### Step 2: Register in the Switch Statement

Add a case to the `Apply` function in the same file:

```go
case "myapp":
    result = myapp(data)
```

### Step 3: Update the CRD Enum Validation

In `api/v1alpha1/oidcclient_types.go`, add the new profile name to the `SecretProfile` enum marker:

```go
// +kubebuilder:validation:Enum=grafana;openwebui;argocd;generic;myapp
SecretProfile string `json:"secretProfile"`
```

### Step 4: Regenerate Manifests

```bash
just generate
```

### Step 5: Add Tests

Add test cases in `internal/profiles/profiles_test.go` to verify the new profile maps keys correctly.

### Step 6: Document the Profile

Create a documentation page at `docs/profiles/myapp.md` describing the keys produced and how to use the profile.

---

## Docker

### Building Images

```bash
# Build with version tag from git
just docker::build

# Build with a custom tag
make docker-build IMG=my-registry/authentik-operator:v1.0.0
```

The image is tagged with both the git-derived version and `latest`.

### Pushing Images

```bash
# Push to GHCR (ghcr.io/kettleofketchup/authentik-operator)
just docker::push
```

You must be logged into the GitHub Container Registry first:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin
```

### Cross-Platform Builds

For multi-architecture images (amd64, arm64):

```bash
make docker-buildx IMG=ghcr.io/kettleofketchup/authentik-operator:v1.0.0
```

---

## Documentation

Documentation is built with [MkDocs Material](https://squidfun.github.io/mkdocs-material/) and managed through `uv` for Python dependencies.

### Local Development

```bash
# Start the dev server with live reload at http://localhost:8000
just docs::serve

# Build static HTML to public/
just docs::build

# Clean generated output
just docs::clean
```

### Adding Pages

1. Create a new `.md` file under `docs/`.
2. Add it to the `nav` section in `mkdocs.yml`.
3. Preview with `just docs::serve`.
