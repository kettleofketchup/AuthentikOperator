# Docker build and push recipes
# Call as: just docker::build, just docker::push, etc.

TOOL_NAME := "AuthentikOperator"
PROJECT_ROOT := env_var("PWD")
VERSION := `git describe --tags --always 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
DATE := `date -u +%Y-%m-%dT%H:%M:%SZ`

DOCKER_REGISTRY := "ghcr.io/kettleofketchup/AuthentikOperator"
DOCKER_TAG := DOCKER_REGISTRY + "/" + TOOL_NAME + ":" + VERSION
DOCKER_LATEST := DOCKER_REGISTRY + "/" + TOOL_NAME + ":latest"
DOCKERFILE := PROJECT_ROOT + "/docker/Dockerfile." + TOOL_NAME

# Login to GitHub Container Registry
login:
    @echo "Logging into GitHub Container Registry..."
    @echo "Use: echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin"

# Build Docker image
build:
    @echo "Building Docker image..."
    cd {{ PROJECT_ROOT }} && docker build \
        --build-arg VERSION={{ VERSION }} \
        --build-arg COMMIT={{ COMMIT }} \
        --build-arg BUILD_DATE={{ DATE }} \
        -t {{ DOCKER_TAG }} \
        -f {{ DOCKERFILE }} .
    docker tag {{ DOCKER_TAG }} {{ DOCKER_LATEST }}
    @echo "Built: {{ DOCKER_TAG }}"

# Push Docker image to registry
push: build
    @echo "Pushing Docker image..."
    docker push {{ DOCKER_TAG }}
    docker push {{ DOCKER_LATEST }}
