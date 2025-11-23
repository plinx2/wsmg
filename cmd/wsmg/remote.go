package main

import (
	"github.com/spf13/cobra"
)

// remoteCmd represents the remote command
var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote repositories",
	Long:  `Manage remote repositories from GitHub, GitLab, and other providers.`,
}

func init() {
	rootCmd.AddCommand(remoteCmd)
}
