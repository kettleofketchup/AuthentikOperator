# Testing recipes
# Call as: just testing::coverage, just testing::bench

TOOL_FOLDER := env_var("PWD") + "/src/{{ tool_name }}"

# Run tests with coverage report
coverage:
    @echo "Running tests with coverage..."
    cd {{ TOOL_FOLDER }} && go test -race -coverprofile=coverage.out ./...
    cd {{ TOOL_FOLDER }} && go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: {{ TOOL_FOLDER }}/coverage.html"

# Run benchmarks
bench:
    @echo "Running benchmarks..."
    cd {{ TOOL_FOLDER }} && go test -bench=. -benchmem ./...
