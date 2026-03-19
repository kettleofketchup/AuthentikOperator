# AuthentikOperator - Go CLI project
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

# Import top-level recipes (merged into root namespace)
import 'just/dev.just'

# Variables
TOOL_NAME := "AuthentikOperator"
PROJECT_NAME := "AuthentikOperator"
TOOL_FOLDER := env_var("PWD") + "/src/" + TOOL_NAME

# List all available recipes
default:
    just --list

# Build the CLI binary (alias for go::build)
[group('dev')]
build:
    just go::build {{ TOOL_NAME }}

# Run tests (alias for go::test)
[group('dev')]
[group('ci')]
test:
    just go::test {{ TOOL_NAME }}

# Run linter (alias for go::lint)
[group('dev')]
[group('ci')]
lint:
    just go::lint {{ TOOL_NAME }}

# Build and run the binary
[group('dev')]
run: build
    ./bin/{{ TOOL_NAME }}
