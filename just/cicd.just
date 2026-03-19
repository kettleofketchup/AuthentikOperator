# CI/CD recipes - platform-agnostic build commands
# Call as: just cicd::lint, just cicd::test, etc.
# These recipes are called by GitLab CI / GitHub Actions

TOOL_NAME := "AuthentikOperator"
TOOL_FOLDER := env_var("PWD") + "/src/" + TOOL_NAME
BIN_DIR := env_var("PWD") + "/bin"

VERSION := `git describe --tags --always 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
DATE := `date -u +%Y-%m-%dT%H:%M:%SZ`

DOCKER_REGISTRY := "ghcr.io/kettleofketchup/AuthentikOperator"
DOCKER_TAG := DOCKER_REGISTRY + "/" + TOOL_NAME + ":" + VERSION

# Run linter
[group('cicd')]
lint:
    @echo "Running linter..."
    cd {{ TOOL_FOLDER }} && golangci-lint run --timeout 5m

# Run tests with race detection and coverage
[group('cicd')]
test:
    @echo "Running tests..."
    cd {{ TOOL_FOLDER }} && go test -race -cover -v ./...

# Build the binary
[group('cicd')]
build:
    @echo "Building binary..."
    @mkdir -p {{ BIN_DIR }}
    cd {{ TOOL_FOLDER }} && go mod tidy && go build -o {{ BIN_DIR }}/{{ TOOL_NAME }} .

# Build docker image
[group('cicd')]
docker tag=DOCKER_TAG:
    @echo "Building Docker image..."
    docker build \
        --build-arg VERSION={{ VERSION }} \
        --build-arg COMMIT={{ COMMIT }} \
        --build-arg BUILD_DATE={{ DATE }} \
        -t {{ tag }} \
        -f docker/Dockerfile.{{ TOOL_NAME }} .

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
