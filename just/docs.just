# Documentation recipes
# Call as: just docs::serve, just docs::build, etc.

# Start MkDocs development server with live reload
serve:
    @echo "Starting docs server with live reload..."
    uv run mkdocs serve

# Build static documentation
build:
    @echo "Building documentation..."
    uv run mkdocs build

# Clean generated documentation
clean:
    rm -rf public/
    rm -rf site/
