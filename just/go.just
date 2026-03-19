# Go build, test, and lint recipes
# Call as: just go::build, just go::test, etc.

# Version information from git
VERSION := `git describe --tags --always 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
DATE := `date -u +%Y-%m-%dT%H:%M:%SZ`

# ldflags for version injection
VERSION_PKG := "github.com/kettleofketchup/AuthentikOperator/src/AuthentikOperator/version"
LDFLAGS := "-s -w -X " + VERSION_PKG + ".Version=" + VERSION + " -X " + VERSION_PKG + ".Commit=" + COMMIT + " -X " + VERSION_PKG + ".BuildDate=" + DATE
TOOL_FOLDER := env_var("PWD") + "/src/AuthentikOperator"
BIN_DIR := env_var("PWD") + "/bin"

# Build the Go binary
build tool: (tidy tool)
    @echo "Building {{ tool }}..."
    @mkdir -p {{ BIN_DIR }}
    cd {{ TOOL_FOLDER }} && go build -trimpath -o {{ BIN_DIR }}/{{ tool }} -ldflags "{{ LDFLAGS }}" .
    @echo "Built: {{ BIN_DIR }}/{{ tool }}"

# Run tests
test tool:
    @echo "Running tests..."
    cd {{ TOOL_FOLDER }} && go test -race -cover -v ./...

# Run linter (formats first)
lint tool: (format tool)
    @echo "Linting..."
    cd {{ TOOL_FOLDER }} && golangci-lint run

# Format code
format tool:
    @echo "Formatting code..."
    cd {{ TOOL_FOLDER }} && go fmt ./...

# Tidy Go modules
tidy tool:
    @echo "Tidying modules..."
    cd {{ TOOL_FOLDER }} && go mod tidy

# Clean build artifacts
clean:
    @echo "Cleaning..."
    go clean
    rm -rf bin/
    rm -rf dist/
