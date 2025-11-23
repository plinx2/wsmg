package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/repo"
	"github.com/plinx2/wsmg/internal/workspace"
	"github.com/spf13/cobra"
)

// WorkspaceCreateOptions represents options for workspace create command
type WorkspaceCreateOptions struct {
	Repos           []string `flag:"repos" short:"r" usage:"Specify repositories to include (comma-separated, skips interactive mode)"`
	BaseBranch      string   `flag:"base-branch" short:"b" default:"" usage:"Base branch for working branches (default: main/master)"`
	NoVSCode        bool     `flag:"no-vscode" default:"false" usage:"Do not create VSCode workspace file"`
	CopyUncommitted bool     `flag:"copy-uncommitted" default:"true" usage:"Copy uncommitted changes and untracked files to worktree"`
}

func newWorkspaceCreateCmd() *cobra.Command {
	// Define options
	opts := &WorkspaceCreateOptions{}

	// Define command
	cmd := &cobra.Command{
		Use:   "create <ticket-name>",
		Short: "Create a workspace",
		Long: `Create a new workspace.

Interactively select repositories, create working branches for each repository,
and place them in the workspace directory using Git worktree.

The workspace directory structure will mirror the repos directory structure:
  workspaces/<ticket>/provider.com/organization/repository`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runWorkspaceCreate(c.Context(), args[0], opts)
		},
	}

	// Bind flags (auto-generated from tags)
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runWorkspaceCreate(ctx context.Context, name string, opts *WorkspaceCreateOptions) error {
	reposDir := config.GetReposDir()

	// Check if workspace already exists
	if _, err := workspaceClient.GetWorkspace(name); err == nil {
		return fmt.Errorf("workspace '%s' already exists", name)
	}

	// Find repositories
	fmt.Fprintf(os.Stderr, "Scanning repositories in %s...\n", reposDir)
	allRepos, err := repoClient.List(repo.ListInput{})
	if err != nil {
		return fmt.Errorf("failed to find repositories: %w", err)
	}

	if len(allRepos) == 0 {
		return fmt.Errorf("no repositories found")
	}

	// Determine repositories to use
	var selectedRepoNames []string
	if len(opts.Repos) > 0 {
		// Use repositories specified via command line
		selectedRepoNames = opts.Repos
	} else {
		// Interactive selection
		selectedRepoNames, err = selectReposInteractive(allRepos, reposDir)
		if err != nil {
			return err
		}
		if len(selectedRepoNames) == 0 {
			fmt.Println("No repositories selected.")
			return nil
		}
	}

	// Display selected repositories
	fmt.Printf("\nSelected repositories: %d\n", len(selectedRepoNames))
	for _, name := range selectedRepoNames {
		fmt.Printf("  - %s\n", name)
	}

	// Confirmation
	if !cli.ConfirmAction("\nCreate workspace?") {
		fmt.Println("Cancelled.")
		return nil
	}

	// Get copy patterns from config
	copyPatterns := config.GetCopyPatterns()

	fmt.Printf("\nCreating workspace: %s\n\n", name)

	// Create workspace
	result, err := workspaceClient.CreateWorkspace(ctx, workspace.CreateWorkspaceInput{
		Name:            name,
		RepoNames:       selectedRepoNames,
		BaseBranch:      opts.BaseBranch,
		CopyPatterns:    copyPatterns,
		NoVSCode:        opts.NoVSCode,
		CopyUncommitted: opts.CopyUncommitted,
	})
	if err != nil {
		return err
	}

	// Display summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Success: %d repositories\n", result.SuccessCount)
	if result.ErrorCount > 0 {
		fmt.Printf("  Error:   %d repositories\n", result.ErrorCount)
	}
	if result.UncommittedFilesCopied > 0 {
		fmt.Printf("  Copied:  %d uncommitted file(s)\n", result.UncommittedFilesCopied)
	}
	fmt.Printf("\nWorkspace created: %s\n", result.WorkspacePath)

	if !opts.NoVSCode {
		fmt.Printf("\nTo open in VSCode:\n")
		fmt.Printf("  code %s/%s.code-workspace\n", result.WorkspacePath, name)
	}

	return nil
}

func selectReposInteractive(allRepos []*repo.Local, baseDir string) ([]string, error) {
	fmt.Println("\nSelect repositories (enter numbers separated by commas, or 'all' for all):")
	fmt.Println()

	// Display repository list
	for i, r := range allRepos {
		branch, _ := r.Branch()
		fmt.Printf("  [%d] %s (branch: %s)\n", i+1, r.RelativePath(baseDir), branch)
	}

	fmt.Print("\nSelection (e.g., 1,3,5 or all): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)

	if input == "all" {
		var allNames []string
		for _, r := range allRepos {
			allNames = append(allNames, r.RelativePath(baseDir))
		}
		return allNames, nil
	}

	// Parse numbers
	indices := parseIndices(input)
	var selected []string

	for _, idx := range indices {
		if idx > 0 && idx <= len(allRepos) {
			selected = append(selected, allRepos[idx-1].RelativePath(baseDir))
		}
	}

	return selected, nil
}

func parseIndices(input string) []int {
	var indices []int
	parts := strings.Split(input, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err == nil {
			indices = append(indices, idx)
		}
	}

	return indices
}
