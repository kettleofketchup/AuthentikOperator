# Cross-compilation release recipes
# Call as: just release::all, just release::linux, etc.

DIST_DIR := env_var("PWD") + "/dist"
VERSION := `git describe --tags --always 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
DATE := `date -u +%Y-%m-%dT%H:%M:%SZ`

BINARY_NAME := "authentik-operator"

# Build for all platforms
all: linux darwin windows
    @echo "Built all platform binaries in dist/"

# Build Linux binaries
linux:
    @echo "Building for Linux..."
    @mkdir -p {{ DIST_DIR }}
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o {{ DIST_DIR }}/{{ BINARY_NAME }}_linux_amd64 ./cmd/
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o {{ DIST_DIR }}/{{ BINARY_NAME }}_linux_arm64 ./cmd/

# Build macOS binaries
darwin:
    @echo "Building for macOS..."
    @mkdir -p {{ DIST_DIR }}
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -o {{ DIST_DIR }}/{{ BINARY_NAME }}_darwin_amd64 ./cmd/
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o {{ DIST_DIR }}/{{ BINARY_NAME }}_darwin_arm64 ./cmd/

# Build Windows binaries
windows:
    @echo "Building for Windows..."
    @mkdir -p {{ DIST_DIR }}
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o {{ DIST_DIR }}/{{ BINARY_NAME }}_windows_amd64.exe ./cmd/
