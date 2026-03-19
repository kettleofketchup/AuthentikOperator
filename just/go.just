# Go build, test, and lint recipes
# Call as: just go::build, just go::test, etc.

# Build the Go binary (delegates to Kubebuilder Makefile)
build:
    @echo "Building operator..."
    make build

# Run tests (delegates to Kubebuilder Makefile for envtest support)
test:
    @echo "Running tests..."
    make test

# Run linter
lint: format
    @echo "Linting..."
    golangci-lint run ./...

# Format code
format:
    @echo "Formatting code..."
    go fmt ./...

# Tidy Go modules
tidy:
    @echo "Tidying modules..."
    go mod tidy

# Clean build artifacts
clean:
    @echo "Cleaning..."
    go clean
    rm -rf bin/
    rm -rf dist/
    rm -rf testbin/
