/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/repo"
	"github.com/plinx2/wsmg/internal/workspace"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// Global clients initialized in PersistentPreRunE
var (
	workspaceClient *workspace.Client
	repoClient      *repo.Client
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wsmg",
	Short: "Workspace Manager - Multi-repository development tool using Git worktree",
	Long: `wsmg (Workspace Manager) streamlines multi-repository development
by leveraging Git worktree for ticket-based workflows.

When working on tasks across multiple repositories, wsmg creates workspaces
for each ticket and manages working branches using Git worktree. This eliminates
the need for branch switching and stash management when working on multiple
tickets in parallel.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize clients if repos/workspaces directories are configured
		reposDir := config.GetReposDir()
		workspacesDir := config.GetWorkspacesDir()

		// Skip client initialization for commands that don't need them (e.g., config commands)
		if reposDir == "" || workspacesDir == "" {
			return nil
		}

		// Initialize repo client
		rc, err := repo.NewClient(reposDir)
		if err != nil {
			return fmt.Errorf("failed to initialize repo client: %w", err)
		}
		repoClient = rc

		// Initialize workspace client
		wc, err := workspace.NewClient(reposDir, workspacesDir)
		if err != nil {
			return fmt.Errorf("failed to initialize workspace client: %w", err)
		}
		workspaceClient = wc

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Configure global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default: ~/.config/wsmg/wsmg.json)")
	rootCmd.PersistentFlags().String("repos-dir", "", "repos directory path")
	rootCmd.PersistentFlags().String("workspaces-dir", "", "workspaces directory path")

	// Bind flags to viper
	viper.BindPFlag("repos", rootCmd.PersistentFlags().Lookup("repos-dir"))
	viper.BindPFlag("workspaces", rootCmd.PersistentFlags().Lookup("workspaces-dir"))

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("wsmg version %s\n", version)
		},
	})

	// Register completion functions (will be called after client initialization)
	cobra.OnInitialize(registerCompletions)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file specified via command line
		viper.SetConfigFile(cfgFile)
	} else {
		// Get home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Set config file path
		configDir := filepath.Join(home, ".config", "wsmg")
		viper.AddConfigPath(configDir)
		viper.SetConfigName("wsmg")
		viper.SetConfigType("json")
	}

	// Read environment variables (prefix: WSMG_)
	viper.SetEnvPrefix("WSMG")
	viper.AutomaticEnv()

	// Set default values
	home, _ := os.UserHomeDir()
	viper.SetDefault("repos", filepath.Join(home, "repos"))
	viper.SetDefault("workspaces", filepath.Join(home, "workspaces"))
	viper.SetDefault("env", []any{})
	viper.SetDefault("remotes", []any{})

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Use default values if config file not found
			// This is normal for first-time usage, so don't print error
		} else {
			// Other errors
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}

	// Load environment variables
	if err := config.LoadEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading environment variables: %v\n", err)
		os.Exit(1)
	}
}
