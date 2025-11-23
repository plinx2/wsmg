package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/remote"
	"github.com/spf13/cobra"
)

// RemoteCloneOptions represents options for remote clone command
type RemoteCloneOptions struct {
	Provider     string   `flag:"provider" short:"p" default:"github" usage:"Provider name (github, gitlab)"`
	Organization string   `flag:"org" short:"o" default:"" usage:"Organization/Group name"`
	Topics       []string `flag:"topic" short:"t" usage:"Filter by topics"`
	Language     string   `flag:"language" short:"l" default:"" usage:"Filter by programming language"`
	Archived     bool     `flag:"archived" default:"false" usage:"Include archived repositories"`
	Interactive  bool     `flag:"interactive" short:"i" default:"false" usage:"Select repositories interactively"`
	UseSSH       bool     `flag:"ssh" default:"false" usage:"Use SSH URL for cloning"`
	Branch       string   `flag:"branch" short:"b" default:"" usage:"Branch to checkout (empty for default)"`
	Depth        int      `flag:"depth" default:"0" usage:"Create a shallow clone with depth (0 for full clone)"`
	DryRun       bool     `flag:"dry-run" default:"false" usage:"Show what would be cloned without actually cloning"`
}

func newRemoteCloneCmd() *cobra.Command {
	opts := &RemoteCloneOptions{}

	cmd := &cobra.Command{
		Use:   "clone [repository]",
		Short: "Clone remote repositories",
		Long:  `Clone one or more repositories from remote providers to the local repos directory`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runRemoteClone(c.Context(), args, opts)
		},
	}

	// Bind flags using tags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runRemoteClone(ctx context.Context, args []string, opts *RemoteCloneOptions) error {
	reposDir := config.GetReposDir()

	// If repository is specified as argument, clone it directly
	if len(args) > 0 {
		return cloneRepositoryFromArg(ctx, args[0], reposDir, opts)
	}

	// Otherwise, use the existing list-based approach
	return cloneRepositoriesFromList(ctx, reposDir, opts)
}

func cloneRepositoryFromArg(ctx context.Context, repoArg string, reposDir string, opts *RemoteCloneOptions) error {
	// Parse repository path: github.com/org/repo or org/repo
	remoteName, owner, repoName, err := parseRepositoryPath(repoArg, opts.Provider)
	if err != nil {
		return fmt.Errorf("invalid repository path: %w", err)
	}

	// Determine provider type from remote configuration
	providerType, err := getProviderTypeFromRemote(remoteName)
	if err != nil {
		return fmt.Errorf("failed to determine provider type for %s: %w", remoteName, err)
	}

	// Get provider client
	providerClient, err := getProvider(providerType)
	if err != nil {
		return err
	}

	// Get repository details
	fmt.Fprintf(os.Stderr, "Fetching repository info from %s...\n", remoteName)
	repo, err := providerClient.GetRepository(ctx, owner, repoName)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	destPath := filepath.Join(reposDir, remoteName, owner, repoName)

	// Check if already cloned
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("repository already exists at %s", destPath)
	}

	// Dry-run mode
	if opts.DryRun {
		fmt.Printf("Dry-run mode: Would clone %s to %s\n", repo.FullName, destPath)
		return nil
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone repository
	fmt.Printf("Cloning %s...\n", repo.FullName)
	if err := providerClient.Clone(ctx, repo, destPath, remote.CloneOptions{
		UseSSH: opts.UseSSH,
		Branch: opts.Branch,
		Depth:  opts.Depth,
	}); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	fmt.Printf("✓ Cloned: %s\n", repo.FullName)
	fmt.Printf("  Path: %s\n", destPath)
	if opts.Branch != "" {
		fmt.Printf("  Branch: %s\n", opts.Branch)
	}

	return nil
}

func cloneRepositoriesFromList(ctx context.Context, reposDir string, opts *RemoteCloneOptions) error {
	// Get provider client
	provider, err := getProvider(opts.Provider)
	if err != nil {
		return err
	}

	// Determine remote name from provider
	remoteName, err := getRemoteNameFromProvider(opts.Provider)
	if err != nil {
		return fmt.Errorf("failed to determine remote name for provider %s: %w", opts.Provider, err)
	}

	// List repositories
	fmt.Fprintf(os.Stderr, "Fetching repositories from %s...\n", opts.Provider)

	repos, err := provider.ListRepositories(ctx, remote.ListOptions{
		Organization: opts.Organization,
		Topics:       opts.Topics,
		Language:     opts.Language,
		Archived:     opts.Archived,
		Type:         "all",
		Sort:         "updated",
		Direction:    "desc",
	})
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found")
		return nil
	}

	// Filter out already cloned repositories
	var toClone []*remote.Repository
	for _, repo := range repos {
		destPath := filepath.Join(reposDir, remoteName, repo.Owner, repo.Name)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			toClone = append(toClone, repo)
		}
	}

	if len(toClone) == 0 {
		fmt.Println("All repositories are already cloned")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d repositories to clone\n\n", len(toClone))

	// Interactive selection
	var selectedRepos []*remote.Repository
	if opts.Interactive {
		selected, err := selectReposForCloneInteractive(toClone)
		if err != nil {
			return err
		}
		selectedRepos = selected
	} else {
		selectedRepos = toClone
	}

	if len(selectedRepos) == 0 {
		fmt.Println("No repositories selected")
		return nil
	}

	// Dry-run mode
	if opts.DryRun {
		fmt.Println("Dry-run mode: The following repositories would be cloned:")
		for _, repo := range selectedRepos {
			destPath := filepath.Join(reposDir, remoteName, repo.Owner, repo.Name)
			fmt.Printf("  %s -> %s\n", repo.FullName, destPath)
		}
		return nil
	}

	// Clone repositories
	fmt.Printf("\nCloning %d repositor(ies)...\n\n", len(selectedRepos))

	successCount := 0
	errorCount := 0

	for i, repo := range selectedRepos {
		fmt.Printf("[%d/%d] Cloning %s...\n", i+1, len(selectedRepos), repo.FullName)

		destPath := filepath.Join(reposDir, remoteName, repo.Owner, repo.Name)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to create directory: %v\n", err)
			errorCount++
			continue
		}

		// Clone repository
		if err := provider.Clone(ctx, repo, destPath, remote.CloneOptions{
			UseSSH: opts.UseSSH,
			Branch: opts.Branch,
			Depth:  opts.Depth,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to clone: %v\n", err)
			errorCount++
			continue
		}

		fmt.Printf("  ✓ Cloned to %s\n", destPath)
		successCount++
	}

	// Summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Success: %d repositories\n", successCount)
	if errorCount > 0 {
		fmt.Printf("  Error:   %d repositories\n", errorCount)
	}

	return nil
}

// parseRepositoryPath parses a repository path string into remote name, owner, and repo name.
// Supports formats:
//   - github.com/org/repo -> remoteName: "github.com", owner: "org", repo: "repo"
//   - my.gitserver.com/org/repo -> remoteName: "my.gitserver.com", owner: "org", repo: "repo"
//   - org/repo -> remoteName: defaultProvider+".com", owner: "org", repo: "repo"
func parseRepositoryPath(repoPath string, defaultProvider string) (remoteName string, owner string, repo string, err error) {
	parts := strings.Split(repoPath, "/")
	if len(parts) == 2 {
		// Format: org/repo - use default provider
		// For backward compatibility, assume github.com if defaultProvider is "github"
		if defaultProvider == "github" {
			remoteName = "github.com"
		} else {
			remoteName = defaultProvider + ".com"
		}
		return remoteName, parts[0], parts[1], nil
	} else if len(parts) == 3 {
		// Format: remote.com/org/repo - remote name can be any hostname
		remoteName = parts[0]
		return remoteName, parts[1], parts[2], nil
	}
	return "", "", "", fmt.Errorf("invalid repository path format: %s (expected: remote.com/org/repo or org/repo)", repoPath)
}

// getProviderTypeFromRemote determines the provider type from remote name using config.
// Returns the provider type (e.g., "github", "gitlab") based on remotes configuration.
func getProviderTypeFromRemote(remoteName string) (string, error) {
	remotes, err := config.GetRemotes()
	if err != nil {
		return "", fmt.Errorf("failed to get remotes config: %w", err)
	}

	// Try to match remote name with configured remotes
	for _, r := range remotes {
		// Compare hostname directly (e.g., "github.com" matches "github.com")
		if r.Host == remoteName {
			// Convert type to lowercase provider name
			providerType := strings.ToLower(r.Type)
			// Handle common variations
			switch providerType {
			case "github":
				return "github", nil
			case "gitlab":
				return "gitlab", nil
			default:
				return providerType, nil
			}
		}
	}

	// Fallback: try to infer from remote name for backward compatibility
	// e.g., "github.com" -> "github"
	if strings.HasSuffix(remoteName, ".com") {
		providerPart := strings.TrimSuffix(remoteName, ".com")
		// Check if it's a known provider
		if providerPart == "github" || providerPart == "gitlab" {
			return providerPart, nil
		}
	}

	return "", fmt.Errorf("no provider configuration found for remote: %s. Please configure it in remotes section", remoteName)
}

// getRemoteNameFromProvider determines the remote name from provider type using config.
// Returns the remote name (e.g., "github.com", "my.gitserver.com") based on remotes configuration.
func getRemoteNameFromProvider(providerType string) (string, error) {
	remotes, err := config.GetRemotes()
	if err != nil {
		return "", fmt.Errorf("failed to get remotes config: %w", err)
	}

	providerTypeLower := strings.ToLower(providerType)

	// Try to find matching remote by provider type
	for _, r := range remotes {
		remoteTypeLower := strings.ToLower(r.Type)
		if remoteTypeLower == providerTypeLower {
			return r.Host, nil
		}
	}

	// Fallback: use default mapping for common providers
	switch providerTypeLower {
	case "github":
		return "github.com", nil
	case "gitlab":
		return "gitlab.com", nil
	default:
		// Default: assume provider.com format
		return providerTypeLower + ".com", nil
	}
}

func selectReposForCloneInteractive(repos []*remote.Repository) ([]*remote.Repository, error) {
	fmt.Println("Select repositories to clone (enter numbers separated by commas, or 'all' for all):")
	fmt.Println()

	// Display repository list
	for i, repo := range repos {
		lang := repo.Language
		if lang == "" {
			lang = "-"
		}
		fmt.Printf("  [%d] %s (%s, ⭐ %d)\n", i+1, repo.FullName, lang, repo.Stars)
	}

	fmt.Print("\nSelection (e.g., 1,3,5 or all): ")
	var input string
	fmt.Scanln(&input)

	input = strings.TrimSpace(input)

	if input == "all" {
		return repos, nil
	}

	// Parse numbers
	indices := parseIndices(input)
	var selected []*remote.Repository

	for _, idx := range indices {
		if idx > 0 && idx <= len(repos) {
			selected = append(selected, repos[idx-1])
		}
	}

	return selected, nil
}

func init() {
	remoteCmd.AddCommand(newRemoteCloneCmd())
}
