package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/repo"
	"github.com/spf13/cobra"
)

// WorkspaceRemoveOptions represents options for workspace remove command
type WorkspaceRemoveOptions struct {
	Repos       []string `flag:"repo" short:"r" usage:"Repository paths to remove (relative to workspace)"`
	Interactive bool     `flag:"interactive" short:"i" default:"false" usage:"Select repositories interactively"`
	Force       bool     `flag:"force" short:"f" default:"false" usage:"Force remove without confirmation"`
}

func newWorkspaceRemoveCmd() *cobra.Command {
	opts := &WorkspaceRemoveOptions{}

	cmd := &cobra.Command{
		Use:   "remove <workspace-name>",
		Short: "Remove repositories from workspace",
		Long:  `Remove one or more repositories from an existing workspace`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runWorkspaceRemove(c.Context(), args[0], opts)
		},
	}

	// Bind flags using tags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	// Register completions
	cmd.ValidArgsFunction = workspaceNameCompletion

	return cmd
}

func runWorkspaceRemove(ctx context.Context, workspaceName string, opts *WorkspaceRemoveOptions) error {
	reposDir := config.GetReposDir()

	// Check if workspace exists
	ws, err := workspaceClient.GetWorkspace(workspaceName)
	if err != nil {
		return workspaceNotFoundError(workspaceName, err)
	}

	if len(ws.Repositories) == 0 {
		return fmt.Errorf("workspace has no repositories")
	}

	// Get repositories to remove
	var reposToRemove []string
	if opts.Interactive {
		// Interactive selection
		selected, err := selectReposToRemoveInteractive(ws.Repositories, reposDir)
		if err != nil {
			return fmt.Errorf("failed to select repositories: %w", err)
		}
		reposToRemove = selected
	} else if len(opts.Repos) > 0 {
		// Use specified repositories
		reposToRemove = opts.Repos
	} else {
		return fmt.Errorf("no repositories specified. Use --repo or --interactive flag")
	}

	if len(reposToRemove) == 0 {
		fmt.Println("No repositories to remove")
		return nil
	}

	// Display repositories to be removed
	fmt.Printf("\nRepositories to remove from workspace '%s':\n", workspaceName)
	for _, repoPath := range reposToRemove {
		fmt.Printf("  - %s\n", repoPath)
	}

	// Confirmation
	if !opts.Force {
		if !cli.ConfirmAction("\nRemove these repositories?") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remove repositories from workspace
	removedCount, err := workspaceClient.RemoveRepositoriesFromWorkspace(workspaceName, reposToRemove, opts.Force)
	if err != nil {
		return fmt.Errorf("failed to remove repositories: %w", err)
	}

	fmt.Printf("\nSuccessfully removed %d repositor(ies) from workspace '%s'\n", removedCount, workspaceName)

	return nil
}

func selectReposToRemoveInteractive(repos []*repo.Local, baseDir string) ([]string, error) {
	fmt.Println("\nSelect repositories to remove (enter numbers separated by commas, or 'all' for all):")
	fmt.Println()

	// Display repository list
	for i, r := range repos {
		branch, _ := r.Branch()
		fmt.Printf("  [%d] %s (branch: %s)\n", i+1, r.RelativePath(baseDir), branch)
	}

	fmt.Print("\nSelection (e.g., 1,3,5 or all): ")
	var input string
	fmt.Scanln(&input)

	input = strings.TrimSpace(input)

	if input == "all" {
		var allPaths []string
		for _, r := range repos {
			allPaths = append(allPaths, r.RelativePath(baseDir))
		}
		return allPaths, nil
	}

	// Parse numbers
	indices := parseIndices(input)
	var selected []string

	for _, idx := range indices {
		if idx > 0 && idx <= len(repos) {
			selected = append(selected, repos[idx-1].RelativePath(baseDir))
		}
	}

	return selected, nil
}

func init() {
	workspaceCmd.AddCommand(newWorkspaceRemoveCmd())
}
