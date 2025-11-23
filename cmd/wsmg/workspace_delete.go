package main

import (
	"context"
	"fmt"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/workspace"
	"github.com/spf13/cobra"
)

// WorkspaceDeleteOptions represents options for workspace delete command
type WorkspaceDeleteOptions struct {
	Force        bool `flag:"force" short:"f" default:"false" usage:"Force delete even with uncommitted changes"`
	KeepBranches bool `flag:"keep-branches" default:"false" usage:"Delete only worktrees, keep branches"`
	DeleteRemote bool `flag:"delete-remote" default:"false" usage:"Delete remote branches as well"`
}

func newWorkspaceDeleteCmd() *cobra.Command {
	// Define options
	opts := &WorkspaceDeleteOptions{}

	// Define command
	cmd := &cobra.Command{
		Use:   "delete <workspace-name>",
		Short: "Delete a workspace",
		Long:  `Delete a workspace, including Git worktrees and branches.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runWorkspaceDelete(c.Context(), args[0], opts)
		},
	}

	// Bind flags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	// Register completions
	cmd.ValidArgsFunction = workspaceNameCompletion

	return cmd
}

func runWorkspaceDelete(ctx context.Context, workspaceName string, opts *WorkspaceDeleteOptions) error {
	// Get workspace info
	ws, err := workspaceClient.GetWorkspace(workspaceName)
	if err != nil {
		return workspaceNotFoundError(workspaceName, err)
	}

	fmt.Printf("Workspace: %s\n", workspaceName)
	fmt.Printf("Path: %s\n", ws.Path)
	fmt.Printf("Repositories: %d\n", len(ws.Repositories))

	if len(ws.Repositories) > 0 {
		fmt.Println("\nRepositories:")
		for _, r := range ws.Repositories {
			branch, _ := r.Branch()
			fmt.Printf("  - %s (branch: %s)\n", r.Name(), branch)
		}
	}

	// Confirmation
	if !opts.Force {
		if !cli.ConfirmAction("\nDelete workspace?") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	fmt.Printf("\nDeleting workspace...\n\n")

	// Delete workspace
	result, err := workspaceClient.DeleteWorkspace(workspace.DeleteWorkspaceInput{
		Name:         workspaceName,
		Force:        opts.Force,
		KeepBranches: opts.KeepBranches,
		DeleteRemote: opts.DeleteRemote,
	})
	if err != nil {
		return err
	}

	// Display summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Processed: %d repositories\n", result.ProcessedCount)
	fmt.Printf("\nWorkspace deleted: %s\n", workspaceName)

	return nil
}
