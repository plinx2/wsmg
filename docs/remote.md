# Remote Repository Management

`wsmg remote` commands allow you to manage remote repositories from GitHub, GitLab, and other providers.

## Configuration

Set your GitHub token in one of the following ways:

### 1. Environment Variable
```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxx"
```

### 2. Configuration File
Add to `~/.config/wsmg/wsmg.json`:
```json
{
  "repos": "/home/user/repos",
  "workspaces": "/home/user/workspaces",
  "github": {
    "token": "ghp_xxxxxxxxxxxxx"
  }
}
```

## Commands

### `wsmg remote list`

List repositories from remote providers.

**Examples:**

```bash
# List all your repositories
wsmg remote list

# List repositories from an organization
wsmg remote list --org mycompany

# Filter by programming language
wsmg remote list --org mycompany --language go

# Filter by topics
wsmg remote list --topic kubernetes --topic golang

# Exclude archived repositories (default)
wsmg remote list --org mycompany

# Include archived repositories
wsmg remote list --org mycompany --archived

# Limit results
wsmg remote list --org mycompany --limit 10

# Output as JSON
wsmg remote list --org mycompany --format json

# Sort by creation date
wsmg remote list --sort created --direction asc
```

### `wsmg remote clone`

Clone one or more repositories from remote providers.

**Examples:**

```bash
# Clone all repositories from an organization (interactive)
wsmg remote clone --org mycompany --interactive

# Clone all Go repositories
wsmg remote clone --org mycompany --language go

# Clone repositories with specific topics
wsmg remote clone --org mycompany --topic backend --topic api

# Dry-run to see what would be cloned
wsmg remote clone --org mycompany --dry-run

# Clone using SSH instead of HTTPS
wsmg remote clone --org mycompany --ssh

# Create shallow clones
wsmg remote clone --org mycompany --depth 1
```

### `wsmg remote sync`

Sync remote repositories with local repos directory. Detects new repositories that exist remotely but not locally.

**Examples:**

```bash
# Check which repositories are missing locally
wsmg remote sync --org mycompany

# Automatically clone missing repositories
wsmg remote sync --org mycompany --auto-clone

# Sync only Go repositories
wsmg remote sync --org mycompany --language go --auto-clone

# Sync with specific topics
wsmg remote sync --org mycompany --topic backend --auto-clone

# Dry-run to see what would be synced
wsmg remote sync --org mycompany --dry-run

# Use SSH for cloning
wsmg remote sync --org mycompany --auto-clone --ssh
```

## Common Workflows

### 1. Initial Setup - Clone All Organization Repositories

```bash
# Review what's available
wsmg remote list --org mycompany

# Clone all repositories interactively
wsmg remote clone --org mycompany --interactive

# Or clone all at once
wsmg remote clone --org mycompany
```

### 2. Keep Repositories Up to Date

```bash
# Check for new repositories daily
wsmg remote sync --org mycompany

# Automatically clone new repositories
wsmg remote sync --org mycompany --auto-clone
```

### 3. Filter by Technology Stack

```bash
# Clone all Go microservices
wsmg remote clone --org mycompany --language go --topic microservice

# Clone all frontend repositories
wsmg remote clone --org mycompany --topic frontend
```

### 4. Working with Multiple Organizations

```bash
# List repositories from different organizations
wsmg remote list --org company1
wsmg remote list --org company2

# Sync multiple organizations
wsmg remote sync --org company1 --auto-clone
wsmg remote sync --org company2 --auto-clone
```

## Directory Structure

Cloned repositories are organized by provider:

```
~/repos/
  └── github.com/
      ├── mycompany/
      │   ├── backend-api/
      │   ├── frontend-app/
      │   └── shared-lib/
      └── another-org/
          └── some-repo/
```

## Tips

1. **Use `--dry-run` first**: Always preview what would be cloned/synced before running the actual operation.

2. **Filter by topics**: Use `--topic` to organize repositories by their purpose or technology.

3. **Automatic syncing**: Set up a cron job to run `wsmg remote sync --auto-clone` daily to keep your local repos in sync.

4. **SSH vs HTTPS**: Use `--ssh` if you prefer SSH authentication over HTTPS tokens.

5. **Shallow clones**: Use `--depth 1` for faster clones if you don't need the full git history.

## Supported Providers

- **GitHub**: ✅ Fully supported
- **GitLab**: 🚧 Coming soon
- **Bitbucket**: 🚧 Coming soon
- **Gitea**: 🚧 Coming soon

## Troubleshooting

### "GitHub token not found"

Make sure you have set the `GITHUB_TOKEN` environment variable or configured it in `wsmg.json`.

### Rate Limiting

GitHub has API rate limits. If you hit the limit:
- Wait for the limit to reset (usually 1 hour)
- Use a personal access token (higher rate limit than unauthenticated requests)
- Use `--limit` to reduce the number of API calls

### Authentication Failed

Ensure your token has the necessary permissions:
- `repo` scope for private repositories
- `read:org` scope for organization repositories
