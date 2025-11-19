package main

import "github.com/spf13/cobra"

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Repository management",
	Long:  `Manage repositories in the repos directory.`,
}

func init() {
	rootCmd.AddCommand(reposCmd)
	reposCmd.AddCommand(newReposListCmd())
	reposCmd.AddCommand(newReposSyncCmd())
}
