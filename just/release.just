# Cross-compilation release recipes
# Call as: just release::all, just release::linux, etc.

TOOL_NAME := "AuthentikOperator"
TOOL_FOLDER := env_var("PWD") + "/src/" + TOOL_NAME
DIST_DIR := env_var("PWD") + "/dist"
VERSION := `git describe --tags --always 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
DATE := `date -u +%Y-%m-%dT%H:%M:%SZ`
VERSION_PKG := "gitlab.lan/AuthentikOperator/src/AuthentikOperator/version"
LDFLAGS := "-s -w -X " + VERSION_PKG + ".Version=" + VERSION + " -X " + VERSION_PKG + ".Commit=" + COMMIT + " -X " + VERSION_PKG + ".BuildDate=" + DATE

# Build for all platforms
all: linux darwin windows
    @echo "Built all platform binaries in dist/"

# Build Linux binaries
linux:
    @echo "Building for Linux..."
    @mkdir -p {{ DIST_DIR }}
    cd {{ TOOL_FOLDER }} && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o {{ DIST_DIR }}/{{ TOOL_NAME }}_linux_amd64 -ldflags "{{ LDFLAGS }}" .
    cd {{ TOOL_FOLDER }} && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o {{ DIST_DIR }}/{{ TOOL_NAME }}_linux_arm64 -ldflags "{{ LDFLAGS }}" .

# Build macOS binaries
darwin:
    @echo "Building for macOS..."
    @mkdir -p {{ DIST_DIR }}
    cd {{ TOOL_FOLDER }} && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -o {{ DIST_DIR }}/{{ TOOL_NAME }}_darwin_amd64 -ldflags "{{ LDFLAGS }}" .
    cd {{ TOOL_FOLDER }} && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o {{ DIST_DIR }}/{{ TOOL_NAME }}_darwin_arm64 -ldflags "{{ LDFLAGS }}" .

# Build Windows binaries
windows:
    @echo "Building for Windows..."
    @mkdir -p {{ DIST_DIR }}
    cd {{ TOOL_FOLDER }} && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o {{ DIST_DIR }}/{{ TOOL_NAME }}_windows_amd64.exe -ldflags "{{ LDFLAGS }}" .
