# wsmg - Workspace Manager

`wsmg` (Workspace Manager) is a command-line tool that streamlines multi-repository development by leveraging Git worktree for ticket-based workflows.

## Overview

When working on development tasks that span multiple repositories, `wsmg` creates workspaces for each ticket (task) and manages working branches using Git worktree. This eliminates the need for branch switching and stash management when working on multiple tickets in parallel.

## Key Features

- **Multi-repository Support**: Create workspaces containing multiple repositories
- **Git Worktree Integration**: Keep default branches in the `repos` directory while managing working branches in the `workspaces` directory
- **VSCode Integration**: Automatically generate `.code-workspace` files for each workspace
- **Remote Repository Management**: Support for GitHub, GitLab, and other Git servers
- **Environment Variable Management**: Configure environment variables per workspace

## Installation

```bash
go install github.com/plinx2/wsmg@latest
```

### Shell Completion

Enable shell completion for faster command input:

```bash
# Bash
wsmg completion bash > /etc/bash_completion.d/wsmg
source /etc/bash_completion.d/wsmg

# Zsh
wsmg completion zsh > "${fpath[1]}/_wsmg"
# Restart your shell

# Fish
wsmg completion fish > ~/.config/fish/completions/wsmg.fish

# PowerShell
wsmg completion powershell > wsmg.ps1
# Add to your PowerShell profile
```

The completion supports:
- Command and subcommand names
- Workspace names
- Repository paths
- Flag options (format, provider, etc.)

## Configuration

Configuration file: `~/.config/wsmg/wsmg.json`

Configuration precedence (highest to lowest):

1. Command-line flags (e.g., `--repos ~/my-repos`)
2. Environment variables (e.g., `WSMG_REPOS=~/my-repos`)
3. Configuration file
4. Default values

```json
{
  "repos": "~/repos",
  "workspaces": "~/workspaces",
  "env": [
    {
      "key": "GITHUB_TOKEN",
      "value": "xxxxxx"
    }
  ],
  "remotes": [
    {
      "type": "GitHub",
      "host": "github.com"
    }
  ],
  "copy_patterns": [
    ".env*",
    ".tool-versions"
  ]
}
```

### Configuration Fields

- `repos`: Directory for cloning repositories (default: `~/repos`)
- `workspaces`: Directory for creating workspaces (default: `~/workspaces`)
- `env`: Environment variables to load automatically when running commands
- `remotes`: Configuration for Git servers to access
- `copy_patterns`: File patterns to copy from source repository to workspace. Patterns are matched against relative paths from the project root. Supports wildcards (`*`) and directory patterns (e.g., `.vscode/*`). The scan depth is limited to 3 levels. Default patterns:
  - `.env*` - Environment files (`.env`, `.env.local`, `.envrc`, etc.) in any directory
  - `.vscode/*` - VSCode editor settings (copies entire directory)
  - `.tool-versions` - asdf version manager
  - `.*-version` - Version files (`.node-version`, `.ruby-version`, `.python-version`, etc.)
  - `.nvmrc` - Node Version Manager
  - `.npmrc` - npm configuration
  - `go.work` - Go workspace file

## Directory Structure

### repos Directory

Maintains the default branch state following the repository path structure of GitHub or GitLab.

```
repos/
├── github.com/
│   ├── {organization}/
│   │   └── {repository}/  # Default branch
│   └── {organization}/
│       └── {repository}/  # Default branch
└── gitlab.com/
    └── {organization}/
        └── {repository}/  # Default branch
```

### workspaces Directory

Creates workspaces per ticket and manages working directories with Git worktree.

```
workspaces/
└── {ticket}/
    ├── github.com/
    │   └── {organization}/
    │       └── {repository}/  # Git worktree
    └── {ticket}.code-workspace  # VSCode workspace configuration
```

## Usage

### Initialize Configuration

```bash
wsmg config init
```

Initialize the configuration file. Recommended to run on first use.

### Show Configuration

```bash
wsmg config show
```

Display current configuration. Shows the actual configuration values used, considering the precedence of command-line flags, environment variables, and configuration file.

### Get/Set Configuration Values

```bash
# Get a configuration value
wsmg config get repos

# Set a configuration value
wsmg config set repos ~/my-repos

# Show configuration file path
wsmg config path

# Edit configuration file
wsmg config edit

# Validate configuration
wsmg config validate [--strict]
```

The `validate` command checks:
- Required fields (repos, workspaces)
- Directory existence and permissions
- Remote host formats
- JSON syntax

Use `--strict` flag to also check for unused keys in the configuration file.

### List Local Repositories

```bash
wsmg repos list [--filter=pattern]
```

Display a list of repositories in the `repos` directory, showing path, branch, last commit, and other information.

### Remote Repository Management

#### List Repositories

```bash
# Basic usage
wsmg remote list

# Filter
wsmg remote list --filter=frontend
wsmg remote list --org=org1

# Show details
wsmg remote list --details

# Cache control
wsmg remote list --no-cache
wsmg remote list --refresh
```

Fetch and display the list of repositories from configured remote servers (GitHub, GitLab, etc.).

#### Clone Repositories

```bash
# Clone a single repository
wsmg remote clone github.com/org1/backend-api

# Use SSH URL
wsmg remote clone github.com/org1/backend-api --ssh

# Specify branch
wsmg remote clone github.com/org1/backend-api --branch=develop

# Interactive selection
wsmg remote clone --interactive
```

Clone remote repositories into the `repos` directory, maintaining the organization structure.

### Workspace Management

#### List Workspaces

```bash
wsmg workspace list [--format=table|json|yaml]
```

Display a list of created workspaces with repository information.

#### Show Workspace Details

```bash
wsmg workspace show <workspace-name> [--format=table|json|yaml]
```

Display detailed information about a specific workspace, including:
- Repository status (clean/dirty)
- Current branch
- Modified and untracked files count
- Last commit information

#### Create Workspace

```bash
# Interactive mode (default)
wsmg workspace create <ticket-name>

# With specific repositories
wsmg workspace create <ticket-name> --repo=github.com/org/repo1 --repo=github.com/org/repo2

# Copy uncommitted changes from original repository
wsmg workspace create <ticket-name> --copy-uncommitted
```

Create a workspace with the specified ticket name. Interactively select repositories, create working branches for each repository, and place them in the workspace directory using Git worktree.

**Automatic Environment File Copying**: When creating a workspace, environment configuration files (`.env*`, `.tool-versions`, `.nvmrc`, etc.) are automatically copied from the source repository to the workspace. This enables seamless development without manual configuration setup.

**Copy Uncommitted Changes**: Use `--copy-uncommitted` flag to copy modified and untracked files from the original repository to the new workspace. Useful when you've started working in the main repository and want to continue in a dedicated workspace.

#### Add Repositories to Workspace

```bash
# Interactive mode
wsmg workspace add <workspace-name> --interactive

# With specific repositories
wsmg workspace add <workspace-name> --repo=github.com/org/repo3

# Copy uncommitted changes
wsmg workspace add <workspace-name> --repo=github.com/org/repo3 --copy-uncommitted
```

Add additional repositories to an existing workspace.

#### Remove Repositories from Workspace

```bash
# Interactive mode
wsmg workspace remove <workspace-name> --interactive

# With specific repositories
wsmg workspace remove <workspace-name> --repo=github.com/org/repo1

# Force removal without confirmation
wsmg workspace remove <workspace-name> --repo=github.com/org/repo1 --force
```

Remove repositories from a workspace. This will delete the worktree and update VSCode configuration.

#### Rename Workspace

```bash
wsmg workspace rename <old-name> <new-name> [--force]
```

Rename a workspace, including:
- Workspace directory
- Git branches in all repositories
- VSCode workspace file

Branches matching the old workspace name will be renamed to the new name.

#### Delete Workspace

```bash
wsmg workspace delete <workspace-name> [--force]
```

Delete the specified workspace, including the removal of Git worktrees.

## License

See [LICENSE](LICENSE).

## Development

For detailed design documentation, see [DESIGN.md](DESIGN.md).
