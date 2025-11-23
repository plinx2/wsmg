# Development Guide

This document describes the coding standards and patterns used in the wsmg project.

## Table of Contents

- [Adding New Commands](#adding-new-commands)
- [Command Structure Pattern](#command-structure-pattern)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Common Mistakes](#common-mistakes)

## Adding New Commands

### Command Structure Pattern

All commands in wsmg **MUST** follow this standardized structure:

#### 1. Options Struct with Tags

```go
// XxxYyyOptions represents options for xxx yyy command
type XxxYyyOptions struct {
    Field1 string   `flag:"field1" short:"f" default:"" usage:"Description of field1"`
    Field2 []string `flag:"field2" short:"F" usage:"Description of field2"`
    Field3 bool     `flag:"field3" default:"false" usage:"Description of field3"`
    Field4 int      `flag:"field4" default:"0" usage:"Description of field4"`
}
```

**Tag Format:**
- `flag:"..."` - Flag name (required)
- `short:"..."` - Short flag (optional, single letter)
- `default:"..."` - Default value (optional)
- `usage:"..."` - Description (required)

#### 2. Command Constructor

```go
func newXxxYyyCmd() *cobra.Command {
    opts := &XxxYyyOptions{}

    cmd := &cobra.Command{
        Use:   "yyy",
        Short: "Brief description",
        Long:  `Detailed description`,
        Args:  cobra.ExactArgs(1), // if positional args required
        RunE: func(c *cobra.Command, args []string) error {
            return runXxxYyy(c.Context(), opts)
        },
    }

    // Bind flags using tags (REQUIRED)
    if err := cli.BindFlags(cmd, opts); err != nil {
        panic(fmt.Sprintf("failed to bind flags: %v", err))
    }

    return cmd
}
```

#### 3. Run Function

```go
func runXxxYyy(ctx context.Context, opts *XxxYyyOptions) error {
    // Implementation
    return nil
}
```

**Function Signature Rules:**
- MUST accept `context.Context` as first parameter
- MUST accept pointer to options struct as second parameter
- MUST return `error`

#### 4. Init Function

```go
func init() {
    xxxCmd.AddCommand(newXxxYyyCmd())
}
```

### Complete Example

```go
package main

import (
    "context"
    "fmt"

    "github.com/plinx2/wsmg/internal/cli"
    "github.com/spf13/cobra"
)

// WorkspaceShowOptions represents options for workspace show command
type WorkspaceShowOptions struct {
    Format string `flag:"format" short:"f" default:"table" usage:"Output format (table, json, yaml)"`
    Verbose bool  `flag:"verbose" short:"v" default:"false" usage:"Show detailed information"`
}

func newWorkspaceShowCmd() *cobra.Command {
    opts := &WorkspaceShowOptions{}

    cmd := &cobra.Command{
        Use:   "show <workspace-name>",
        Short: "Show workspace details",
        Long:  `Display detailed information about a workspace`,
        Args:  cobra.ExactArgs(1),
        RunE: func(c *cobra.Command, args []string) error {
            return runWorkspaceShow(c.Context(), args[0], opts)
        },
    }

    if err := cli.BindFlags(cmd, opts); err != nil {
        panic(fmt.Sprintf("failed to bind flags: %v", err))
    }

    return cmd
}

func runWorkspaceShow(ctx context.Context, workspaceName string, opts *WorkspaceShowOptions) error {
    // Implementation
    fmt.Printf("Showing workspace: %s\n", workspaceName)
    return nil
}

func init() {
    workspaceCmd.AddCommand(newWorkspaceShowCmd())
}
```

## Coding Standards

### Language

- **All comments, error messages, and user-facing text MUST be in English**
- Variable names MUST be in English
- Function names MUST be in English

### Naming Conventions

#### Types

```go
// Good
type WorkspaceCreateOptions struct { }
type RemoteListOptions struct { }

// Bad
type CreateWorkspaceOptions struct { } // Wrong order
type remote_list_options struct { }    // Wrong case
```

#### Functions

```go
// Good
func runWorkspaceCreate(ctx context.Context, opts *WorkspaceCreateOptions) error
func newWorkspaceCreateCmd() *cobra.Command

// Bad
func WorkspaceCreate(ctx context.Context, opts *WorkspaceCreateOptions) error  // Should not be exported
func createWorkspace(ctx context.Context, opts *WorkspaceCreateOptions) error  // Missing 'run' prefix
```

#### Variables

```go
// Good
var workspaceClient *workspace.Client
var repoClient *repo.Client

// Bad
var client *workspace.Client  // Too generic
var wsClient *workspace.Client  // Avoid abbreviations
```

### Package Structure

```
cmd/wsmg/
  ├── root.go              # Root command and global initialization
  ├── workspace.go         # Workspace subcommand group
  ├── workspace_create.go  # workspace create command
  ├── workspace_delete.go  # workspace delete command
  ├── workspace_list.go    # workspace list command
  ├── repos.go             # Repos subcommand group
  ├── repos_list.go        # repos list command
  └── ...

internal/
  ├── cli/                 # CLI utilities
  │   ├── confirm.go
  │   └── flags.go
  ├── config/              # Configuration management
  ├── repo/                # Repository operations
  │   ├── local.go
  │   └── ...
  ├── workspace/           # Workspace operations
  │   └── workspace.go
  └── remote/              # Remote provider integration
      ├── interface.go
      ├── types.go
      └── github/
          └── client.go
```

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("failed to create workspace: %w", err)
}

// Bad
if err != nil {
    return err  // Lost context
}

if err != nil {
    panic(err)  // Don't panic in command code
}
```

### Client Initialization

```go
// Good - Use global clients initialized in root.PersistentPreRunE
func runWorkspaceCreate(ctx context.Context, opts *WorkspaceCreateOptions) error {
    result, err := workspaceClient.CreateWorkspace(ctx, ...)
}

// Bad - Don't create clients in each command
func runWorkspaceCreate(ctx context.Context, opts *WorkspaceCreateOptions) error {
    client, err := workspace.NewClient(...)  // Don't do this
}
```

## Common Mistakes

### ❌ DON'T: Manual Flag Binding

```go
// WRONG
func newWorkspaceCreateCmd() *cobra.Command {
    opts := &WorkspaceCreateOptions{}
    cmd := &cobra.Command{...}

    cmd.Flags().StringVarP(&opts.Repos, "repos", "r", "", "...")
    cmd.Flags().StringVar(&opts.BaseBranch, "base-branch", "", "...")

    return cmd
}
```

### ✅ DO: Tag-based Flag Binding

```go
// CORRECT
type WorkspaceCreateOptions struct {
    Repos      []string `flag:"repos" short:"r" usage:"..."`
    BaseBranch string   `flag:"base-branch" default:"" usage:"..."`
}

func newWorkspaceCreateCmd() *cobra.Command {
    opts := &WorkspaceCreateOptions{}
    cmd := &cobra.Command{...}

    if err := cli.BindFlags(cmd, opts); err != nil {
        panic(fmt.Sprintf("failed to bind flags: %v", err))
    }

    return cmd
}
```

### ❌ DON'T: Create Clients in Commands

```go
// WRONG
func runReposList(ctx context.Context, opts *ReposListOptions) error {
    client, err := repo.NewClient(reposDir)
    if err != nil {
        return err
    }
    repos, err := client.List(...)
}
```

### ✅ DO: Use Global Clients

```go
// CORRECT
func runReposList(ctx context.Context, opts *ReposListOptions) error {
    repos, err := repoClient.List(...)
}
```

### ❌ DON'T: Japanese Comments/Messages

```go
// WRONG
// ワークスペースを作成します
func runWorkspaceCreate(...) error {
    fmt.Println("ワークスペースを作成しています...")
}
```

### ✅ DO: English Comments/Messages

```go
// CORRECT
// runWorkspaceCreate creates a new workspace
func runWorkspaceCreate(...) error {
    fmt.Println("Creating workspace...")
}
```

## Testing

### Unit Tests

Test files should be placed next to the code they test:

```
internal/workspace/
  ├── workspace.go
  └── workspace_test.go
```

### Integration Tests

(To be defined)

## Git Workflow

1. Create a feature branch from `main`
2. Make changes following this guide
3. Ensure all tests pass
4. Submit pull request
5. Address review comments
6. Merge after approval

## Questions?

If you're unsure about a pattern or convention:

1. Look at existing code in the same area
2. Check this DEVELOPMENT.md
3. Ask in pull request comments

## Checklist for New Commands

Before submitting a PR with a new command:

- [ ] Options struct uses struct tags
- [ ] Uses `cli.BindFlags` (not manual `cmd.Flags()`)
- [ ] Run function signature: `func runXxx(ctx context.Context, opts *XxxOptions) error`
- [ ] All comments and messages in English
- [ ] Uses global clients (not creating new clients)
- [ ] Added to parent command's `init()`
- [ ] Help text is clear and complete
- [ ] Examples added to relevant docs
