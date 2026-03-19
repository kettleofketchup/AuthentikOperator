# Claude Code Instructions for AuthentikOperator

## Project Overview

AuthentikOperator is a Go CLI tool built with Cobra.

## Architecture

```
src/AuthentikOperator/
├── cmd/           # CLI commands (Cobra)
├── internal/      # Private packages
└── version/       # Build-time version info
```

## Key Technologies

- **Language**: Go 1.23+
- **CLI**: Cobra
- **Config**: Viper (YAML/JSON/TOML)

## Build System

The build system uses [just](https://github.com/casey/just) with modular `.just` files in `just/`.

Modules are namespaced with `::` syntax (e.g., `just docker::build`).

### Quick Reference

```sh
# Build (top-level aliases)
just build                  # Build AuthentikOperator binary → bin/AuthentikOperator
just test                   # Run all tests
just lint                   # Format + lint
just run                    # Build and run

# Module commands
just go::build AuthentikOperator    # Build with explicit tool name
just go::format AuthentikOperator   # Format code only
just go::tidy AuthentikOperator     # Tidy modules
just go::clean              # Remove bin/ and dist/
```

### Documentation

```sh
just docs::serve            # Serve docs locally (localhost:8000)
just docs::build            # Build docs → public/
```

### Docker

```sh
just docker::build          # Build Docker image
just docker::push           # Push to registry
```

### Release

```sh
just release::all           # Build for all platforms → dist/
just release::linux         # Linux only (amd64 + arm64)
```

## Code Conventions

### Adding Commands

Create new commands in `src/AuthentikOperator/cmd/`:

```go
var exampleCmd = &cobra.Command{
    Use:   "example",
    Short: "Example command",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(exampleCmd)
}
```

### Error Handling

Use `RunE` instead of `Run` for proper error handling:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if err := doSomething(); err != nil {
        return fmt.Errorf("failed to do something: %w", err)
    }
    return nil
}
```

### Configuration

Use Viper for configuration:

```go
viper.GetString("key")
viper.GetInt("port")
viper.GetBool("enabled")
```

## Worktree Convention

Use `.worktrees/` for git worktrees:

```sh
git worktree add .worktrees/feature-name -b feature/feature-name
```
