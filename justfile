# AuthentikOperator - Kubernetes Operator
# Run `just --list` to see all available recipes

set quiet
set dotenv-load

# Modules - call as `just module::recipe`
mod go 'just/go.just'
mod docs 'just/docs.just'
mod docker 'just/docker.just'
mod release 'just/release.just'
mod compose 'just/compose.just'
mod certs 'just/certs.just'
mod testing 'just/testing.just'
mod cicd 'just/cicd.just'
mod copier 'just/copier.just'
mod chart 'just/chart.just'

# Import top-level recipes (merged into root namespace)
import 'just/dev.just'

# Variables
TOOL_NAME := "manager"

# List all available recipes
default:
    just --list

# Build the operator binary (wraps make build)
[group('dev')]
build:
    make build

# Run tests (wraps make test — includes envtest setup)
[group('dev')]
[group('ci')]
test:
    make test

# Run linter
[group('dev')]
[group('ci')]
lint:
    just go::lint

# Generate CRD manifests and deepcopy methods
[group('dev')]
generate:
    make manifests generate

# Build and run locally (outside cluster)
[group('dev')]
run: build
    ./bin/{{ TOOL_NAME }}
