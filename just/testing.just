# Testing recipes
# Call as: just testing::coverage, just testing::bench

# Run tests with coverage report (wraps make test)
coverage:
    @echo "Running tests with coverage..."
    make test
    @echo "Coverage output in cover.out"

# Run benchmarks
bench:
    @echo "Running benchmarks..."
    go test -bench=. -benchmem ./...
