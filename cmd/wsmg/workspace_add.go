package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/repo"
	"github.com/spf13/cobra"
)

// WorkspaceAddOptions represents options for workspace add command
type WorkspaceAddOptions struct {
	Interactive     bool
	Repos           []string
	CopyUncommitted bool
}

func newWorkspaceAddCmd() *cobra.Command {
	opts := &WorkspaceAddOptions{}

	cmd := &cobra.Command{
		Use:   "add <workspace-name>",
		Short: "Add repositories to an existing workspace",
		Long:  `Add one or more repositories to an existing workspace`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceAdd(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Interactive, "interactive", "i", false, "Select repositories interactively")
	cmd.Flags().StringSliceVarP(&opts.Repos, "repo", "r", []string{}, "Repository paths to add (relative to repos directory)")
	cmd.Flags().BoolVar(&opts.CopyUncommitted, "copy-uncommitted", false, "Copy uncommitted changes and untracked files to worktree")

	return cmd
}

func runWorkspaceAdd(ctx context.Context, workspaceName string, opts *WorkspaceAddOptions) error {
	reposDir := config.GetReposDir()
	workspacesDir := config.GetWorkspacesDir()

	// Check if workspace exists
	if _, err := workspaceClient.GetWorkspace(workspaceName); err != nil {
		return workspaceNotFoundError(workspaceName, err)
	}

	workspacePath := filepath.Join(workspacesDir, workspaceName)

	// Get all available repositories
	allRepos, err := repoClient.List(repo.ListInput{})
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(allRepos) == 0 {
		return fmt.Errorf("no repositories found in %s", reposDir)
	}

	// Get repository paths to add
	var repoPaths []string
	if opts.Interactive {
		// Interactive selection
		selected, err := selectReposInteractive(allRepos, reposDir)
		if err != nil {
			return fmt.Errorf("failed to select repositories: %w", err)
		}
		repoPaths = selected
	} else if len(opts.Repos) > 0 {
		// Use specified repositories
		repoPaths = opts.Repos
	} else {
		return fmt.Errorf("no repositories specified. Use --repo or --interactive flag")
	}

	if len(repoPaths) == 0 {
		fmt.Println("No repositories to add")
		return nil
	}

	// Add repositories to workspace
	copiedFiles, err := workspaceClient.AddRepositoriesToWorkspace(workspaceName, repoPaths, opts.CopyUncommitted)
	if err != nil {
		return fmt.Errorf("failed to add repositories: %w", err)
	}

	fmt.Printf("Successfully added %d repositor(ies) to workspace '%s'\n", len(repoPaths), workspaceName)
	if copiedFiles > 0 {
		fmt.Printf("Copied %d uncommitted file(s)\n", copiedFiles)
	}
	fmt.Printf("Workspace: %s\n", workspacePath)

	return nil
}

func init() {
	workspaceCmd.AddCommand(newWorkspaceAddCmd())
}
