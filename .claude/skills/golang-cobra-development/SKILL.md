---
name: golang-cobra-development
description: Use when building CLI commands, adding flags, creating subcommands, or integrating Viper configuration in this Go project
---

# Go Cobra CLI Development

## Overview

Patterns for building CLI applications with Cobra and Viper in AuthentikOperator.

## Project Structure

```
src/AuthentikOperator/
├── main.go              # Entry point
├── cmd/
│   ├── root.go          # Root command, global flags
│   └── version.go       # Version subcommand
├── internal/            # Private packages
└── version/
    └── version.go       # Build-time version variables
```

## Adding a New Command

Create `cmd/<command>.go`:

```go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var exampleCmd = &cobra.Command{
    Use:   "example",
    Short: "Short description",
    Long:  `Longer description.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        name, _ := cmd.Flags().GetString("name")
        fmt.Printf("Hello, %s!\n", name)
        return nil
    },
}

func init() {
    rootCmd.AddCommand(exampleCmd)
    exampleCmd.Flags().StringP("name", "n", "world", "Name to greet")
}
```

## Subcommands

```go
var parentCmd = &cobra.Command{Use: "parent", Short: "Parent command"}
var childCmd = &cobra.Command{
    Use:   "child",
    Short: "Child command",
    RunE: func(cmd *cobra.Command, args []string) error {
        return nil
    },
}

func init() {
    rootCmd.AddCommand(parentCmd)
    parentCmd.AddCommand(childCmd)
}
```

## Flags

| Type | Method | Example |
|------|--------|---------|
| Persistent | `PersistentFlags()` | Available to command + subcommands |
| Local | `Flags()` | Only this command |
| Required | `MarkFlagRequired()` | Must be provided |

```go
// Persistent (inherited by subcommands)
rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

// Local (only this command)
exampleCmd.Flags().BoolP("verbose", "v", false, "verbose output")

// Required
exampleCmd.MarkFlagRequired("name")
```

## Viper Integration

```go
func init() {
    exampleCmd.Flags().String("server", "localhost", "server address")
    viper.BindPFlag("server", exampleCmd.Flags().Lookup("server"))
}

func run(cmd *cobra.Command, args []string) error {
    // Checks: flag → env → config file → default
    server := viper.GetString("server")
    return nil
}
```

## Error Handling

Always use `RunE` for proper error handling:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if err := doThing(); err != nil {
        return fmt.Errorf("failed: %w", err)
    }
    return nil
}
```

## Version Injection

Build with ldflags:

```sh
just build  # Injects version.Version, version.Commit, version.BuildDate
```

## Quick Reference

| Task | Code |
|------|------|
| Add command | `rootCmd.AddCommand(cmd)` |
| String flag | `cmd.Flags().StringP("name", "n", "default", "help")` |
| Bool flag | `cmd.Flags().BoolP("verbose", "v", false, "help")` |
| Get flag | `cmd.Flags().GetString("name")` |
| Viper get | `viper.GetString("key")` |
| Required | `cmd.MarkFlagRequired("name")` |

## References

- [Cobra Documentation](https://cobra.dev/)
- [Cobra User Guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md)
- [Viper Documentation](https://github.com/spf13/viper)
