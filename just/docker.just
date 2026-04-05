# Docker build and push recipes
# Call as: just docker::build, just docker::push, etc.

PROJECT_ROOT := env_var("PWD")
VERSION := `git describe --tags --always 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
DATE := `date -u +%Y-%m-%dT%H:%M:%SZ`

DOCKER_REGISTRY := "ghcr.io/kettleofketchup"
IMAGE_NAME := "authentik-operator"
DOCKER_TAG := DOCKER_REGISTRY + "/" + IMAGE_NAME + ":" + VERSION
DOCKER_LATEST := DOCKER_REGISTRY + "/" + IMAGE_NAME + ":latest"

# Login to GitHub Container Registry
login:
    @echo "Logging into GitHub Container Registry..."
    @echo "Use: echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin"

# Build Docker image
build:
    @echo "Building Docker image..."
    cd {{ PROJECT_ROOT }} && docker build \
        -t {{ DOCKER_TAG }} \
        -f Dockerfile .
    docker tag {{ DOCKER_TAG }} {{ DOCKER_LATEST }}
    @echo "Built: {{ DOCKER_TAG }}"

# Run full test suite in Docker (includes envtest controller tests)
test:
    @echo "Building test image..."
    cd {{ PROJECT_ROOT }} && docker build -f Dockerfile.test -t {{ IMAGE_NAME }}-test .
    @echo "Running tests..."
    docker run --rm {{ IMAGE_NAME }}-test

# Push Docker image to registry
push: build
    @echo "Pushing Docker image..."
    docker push {{ DOCKER_TAG }}
    docker push {{ DOCKER_LATEST }}
