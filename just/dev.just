# Development recipes

# Bootstrap the development environment
dev:
    @echo "Setting up development environment..."
    uv sync
    @echo "Done! Development environment ready."
    @echo ""
    @echo "Available commands:"
    @echo "  just build        Build the operator binary (bin/manager)"
    @echo "  just test         Run tests (including envtest)"
    @echo "  just lint         Run linter"
    @echo "  just generate     Regenerate CRDs and deepcopy"
    @echo "  just docs::serve  Start docs dev server"
