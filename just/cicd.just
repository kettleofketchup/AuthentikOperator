# CI/CD recipes - platform-agnostic build commands
# Call as: just cicd::lint, just cicd::test, etc.
# These recipes are called by GitLab CI / GitHub Actions

BIN_DIR := env_var("PWD") + "/bin"

VERSION := `git describe --tags --always 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
DATE := `date -u +%Y-%m-%dT%H:%M:%SZ`

DOCKER_REGISTRY := "ghcr.io/kettleofketchup"
IMAGE_NAME := "authentik-operator"
DOCKER_TAG := DOCKER_REGISTRY + "/" + IMAGE_NAME + ":" + VERSION

# Run linter
[group('cicd')]
lint:
    @echo "Running linter..."
    golangci-lint run --timeout 5m ./...

# Run tests with envtest (wraps make test)
[group('cicd')]
test:
    @echo "Running tests..."
    make test

# Build the binary
[group('cicd')]
build:
    @echo "Building binary..."
    make build

# Build docker image
[group('cicd')]
docker tag=DOCKER_TAG:
    @echo "Building Docker image..."
    docker build \
        -t {{ tag }} \
        -f Dockerfile .

# Push docker image
[group('cicd')]
docker-push tag=DOCKER_TAG: (docker tag)
    @echo "Pushing Docker image..."
    docker push {{ tag }}

# Build documentation to public/
[group('cicd')]
pages:
    @echo "Building documentation..."
    uv sync
    uv run mkdocs build
