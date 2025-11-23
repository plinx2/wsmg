package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/remote"
	"github.com/spf13/cobra"
)

// RemoteSyncOptions represents options for remote sync command
type RemoteSyncOptions struct {
	Provider     string   `flag:"provider" short:"p" default:"github" usage:"Provider name (github, gitlab)"`
	Organization string   `flag:"org" short:"o" default:"" usage:"Organization/Group name"`
	Topics       []string `flag:"topic" short:"t" usage:"Filter by topics"`
	Language     string   `flag:"language" short:"l" default:"" usage:"Filter by programming language"`
	AutoClone    bool     `flag:"auto-clone" default:"false" usage:"Automatically clone new repositories"`
	UseSSH       bool     `flag:"ssh" default:"false" usage:"Use SSH URL for cloning"`
	Depth        int      `flag:"depth" default:"0" usage:"Create a shallow clone with depth (0 for full clone)"`
	DryRun       bool     `flag:"dry-run" default:"false" usage:"Show what would be synced without actually syncing"`
}

func newRemoteSyncCmd() *cobra.Command {
	opts := &RemoteSyncOptions{}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync remote repositories with local",
		Long: `Sync remote repositories with local repos directory.
Detects new repositories that exist remotely but not locally.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runRemoteSync(c.Context(), opts)
		},
	}

	// Bind flags using tags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	// Register completions
	cmd.RegisterFlagCompletionFunc("provider", providerCompletion)

	return cmd
}

func runRemoteSync(ctx context.Context, opts *RemoteSyncOptions) error {
	reposDir := config.GetReposDir()

	// Get provider client
	provider, err := getProvider(opts.Provider)
	if err != nil {
		return err
	}

	// List remote repositories
	fmt.Fprintf(os.Stderr, "Fetching repositories from %s...\n", opts.Provider)

	remoteRepos, err := provider.ListRepositories(ctx, remote.ListOptions{
		Organization: opts.Organization,
		Topics:       opts.Topics,
		Language:     opts.Language,
		Archived:     false, // Don't sync archived repos by default
		Type:         "all",
		Sort:         "updated",
		Direction:    "desc",
	})
	if err != nil {
		return fmt.Errorf("failed to list remote repositories: %w", err)
	}

	if len(remoteRepos) == 0 {
		fmt.Println("No remote repositories found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d remote repositories\n", len(remoteRepos))

	// Check which repositories are missing locally
	var missingRepos []*remote.Repository
	var existingRepos []*remote.Repository

	for _, repo := range remoteRepos {
		destPath := filepath.Join(reposDir, opts.Provider+".com", repo.Owner, repo.Name)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			missingRepos = append(missingRepos, repo)
		} else {
			existingRepos = append(existingRepos, repo)
		}
	}

	// Display summary
	fmt.Printf("\nSync Status:\n")
	fmt.Printf("  Remote:   %d repositories\n", len(remoteRepos))
	fmt.Printf("  Local:    %d repositories\n", len(existingRepos))
	fmt.Printf("  Missing:  %d repositories\n", len(missingRepos))

	if len(missingRepos) == 0 {
		fmt.Println("\n✓ All remote repositories are already cloned locally")
		return nil
	}

	// Display missing repositories
	fmt.Println("\nMissing repositories:")
	for _, repo := range missingRepos {
		lang := repo.Language
		if lang == "" {
			lang = "-"
		}
		fmt.Printf("  - %s (%s, ⭐ %d)\n", repo.FullName, lang, repo.Stars)
	}

	// Dry-run mode
	if opts.DryRun {
		fmt.Println("\nDry-run mode: Run with --auto-clone to clone missing repositories")
		return nil
	}

	// Auto-clone mode
	if !opts.AutoClone {
		fmt.Println("\nTo clone missing repositories, run with --auto-clone flag")
		return nil
	}

	// Clone missing repositories
	fmt.Printf("\nCloning %d missing repositor(ies)...\n\n", len(missingRepos))

	successCount := 0
	errorCount := 0

	for i, repo := range missingRepos {
		fmt.Printf("[%d/%d] Cloning %s...\n", i+1, len(missingRepos), repo.FullName)

		destPath := filepath.Join(reposDir, opts.Provider+".com", repo.Owner, repo.Name)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to create directory: %v\n", err)
			errorCount++
			continue
		}

		// Clone repository
		if err := provider.Clone(ctx, repo, destPath, remote.CloneOptions{
			UseSSH: opts.UseSSH,
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
	fmt.Printf("\nSync Summary:\n")
	fmt.Printf("  Success: %d repositories\n", successCount)
	if errorCount > 0 {
		fmt.Printf("  Error:   %d repositories\n", errorCount)
	}

	return nil
}

func init() {
	remoteCmd.AddCommand(newRemoteSyncCmd())
}
