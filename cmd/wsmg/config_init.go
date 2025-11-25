package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
)

// ConfigInitOptions represents options for config init command
type ConfigInitOptions struct {
	Force       bool `flag:"force" short:"f" default:"false" usage:"Overwrite existing config file without confirmation"`
	Interactive bool `flag:"interactive" short:"i" default:"false" usage:"Enter configuration values interactively"`
}

func newConfigInitCmd() *cobra.Command {
	// Define options
	opts := &ConfigInitOptions{}

	// Define command
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		Long:  `Initialize configuration file. If a config file already exists, prompts for confirmation to overwrite.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigInit(c.Context(), opts)
		},
	}

	// Bind flags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigInit(ctx context.Context, opts *ConfigInitOptions) error {
	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Config directory path
	configDir := filepath.Join(home, ".config", "wsmg")
	configFile := filepath.Join(configDir, "wsmg.json")

	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil {
		if !opts.Force {
			fmt.Printf("Configuration file already exists: %s\n", configFile)
			if !cli.ConfirmAction("Overwrite?") {
				fmt.Println("Cancelled.")
				return nil
			}
		}
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default configuration
	config := map[string]any{
		"repos":      filepath.Join(home, "repos"),
		"workspaces": filepath.Join(home, "workspaces"),
		"env":        []any{},
		"remotes":    []any{},
	}

	// Interactive mode
	if opts.Interactive {
		fmt.Println("\nEnter configuration values (press Enter for default):")

		// repos directory
		fmt.Printf("repos directory [%s]: ", config["repos"])
		var reposInput string
		fmt.Scanln(&reposInput)
		if reposInput != "" {
			config["repos"] = reposInput
		}

		// workspaces directory
		fmt.Printf("workspaces directory [%s]: ", config["workspaces"])
		var workspacesInput string
		fmt.Scanln(&workspacesInput)
		if workspacesInput != "" {
			config["workspaces"] = workspacesInput
		}
	}

	// Write to JSON file
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate JSON: %w", err)
	}

	if err := os.WriteFile(configFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\nConfiguration file created: %s\n\n", configFile)
	fmt.Println("Default configuration:")
	fmt.Printf("  repos:      %s\n", config["repos"])
	fmt.Printf("  workspaces: %s\n", config["workspaces"])

	return nil
}
