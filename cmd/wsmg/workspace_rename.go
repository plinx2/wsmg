package main

import (
	"context"
	"fmt"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
)

// WorkspaceRenameOptions represents options for workspace rename command
type WorkspaceRenameOptions struct {
	Force bool `flag:"force" short:"f" default:"false" usage:"Force rename without confirmation"`
}

func newWorkspaceRenameCmd() *cobra.Command {
	opts := &WorkspaceRenameOptions{}

	cmd := &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a workspace",
		Long: `Rename a workspace, including:
  - Workspace directory
  - Git branches in all repositories
  - VSCode workspace file`,
		Args: cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			return runWorkspaceRename(c.Context(), args[0], args[1], opts)
		},
	}

	// Bind flags using tags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runWorkspaceRename(ctx context.Context, oldName, newName string, opts *WorkspaceRenameOptions) error {
	// Check if old workspace exists
	oldWs, err := workspaceClient.GetWorkspace(oldName)
	if err != nil {
		return workspaceNotFoundError(oldName, err)
	}

	// Check if new workspace already exists
	if _, err := workspaceClient.GetWorkspace(newName); err == nil {
		return fmt.Errorf("workspace '%s' already exists", newName)
	}

	// Display what will be renamed
	fmt.Printf("Renaming workspace '%s' to '%s'\n\n", oldName, newName)
	fmt.Printf("The following will be renamed:\n")
	fmt.Printf("  - Workspace directory\n")
	fmt.Printf("  - Branches in %d repositor(ies)\n", len(oldWs.Repositories))
	fmt.Printf("  - VSCode workspace file\n\n")

	// Show affected repositories
	if len(oldWs.Repositories) > 0 {
		fmt.Println("Affected repositories:")
		for _, r := range oldWs.Repositories {
			branch, _ := r.Branch()
			fmt.Printf("  - %s (branch: %s -> %s)\n", r.Name(), branch, newName)
		}
		fmt.Println()
	}

	// Confirmation
	if !opts.Force {
		if !cli.ConfirmAction("Proceed with rename?") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Perform rename
	fmt.Println("\nRenaming workspace...")
	if err := workspaceClient.RenameWorkspace(oldName, newName); err != nil {
		return fmt.Errorf("failed to rename workspace: %w", err)
	}

	fmt.Printf("\n✓ Successfully renamed workspace '%s' to '%s'\n", oldName, newName)

	return nil
}

func init() {
	workspaceCmd.AddCommand(newWorkspaceRenameCmd())
}
