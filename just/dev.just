# Development recipes

# Bootstrap the development environment
dev:
    @echo "Setting up development environment..."
    uv sync
    @echo "Done! Development environment ready."
    @echo ""
    @echo "Available commands:"
    @echo "  just build        Build the binary"
    @echo "  just test         Run tests"
    @echo "  just lint         Run linter"
    @echo "  just run          Build and run"
    @echo "  just docs-serve   Start docs dev server"
