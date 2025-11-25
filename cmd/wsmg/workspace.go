package main

import "github.com/spf13/cobra"

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Workspace management",
	Long:  `Create and manage workspaces.`,
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
	workspaceCmd.AddCommand(newWorkspaceListCmd())
	workspaceCmd.AddCommand(newWorkspaceCreateCmd())
	workspaceCmd.AddCommand(newWorkspaceDeleteCmd())
}
