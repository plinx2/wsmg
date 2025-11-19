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
      "url": "https://github.com",
      "organizations": []
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
```

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

### List Workspaces

```bash
wsmg workspace list
```

Display a list of created workspaces.

### Create Workspace

```bash
wsmg workspace create <ticket-name>
```

Create a workspace with the specified ticket name. Interactively select repositories, create working branches for each repository, and place them in the workspace directory using Git worktree.

**Automatic Environment File Copying**: When creating a workspace, environment configuration files (`.env*`, `.tool-versions`, `.nvmrc`, etc.) are automatically copied from the source repository to the workspace. This enables seamless development without manual configuration setup.

### Delete Workspace

```bash
wsmg workspace delete <ticket-name>
```

Delete the specified workspace, including the removal of Git worktrees.

## License

See [LICENSE](LICENSE).

## Development

For detailed design documentation, see [DESIGN.md](DESIGN.md).
