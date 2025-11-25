package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigUnsetOptions represents options for config unset command
type ConfigUnsetOptions struct {
	Force bool `flag:"force" short:"f" default:"false" usage:"Delete without confirmation"`
}

func newConfigUnsetCmd() *cobra.Command {
	// Define options
	opts := &ConfigUnsetOptions{}

	// Define command
	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Delete a configuration value",
		Long:  `Delete the specified key from the configuration file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigUnset(c.Context(), args[0], opts)
		},
	}

	// Bind flags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigUnset(ctx context.Context, key string, opts *ConfigUnsetOptions) error {
	// Check required keys
	requiredKeys := []string{"repos", "workspaces"}
	for _, reqKey := range requiredKeys {
		if key == reqKey {
			return fmt.Errorf("cannot unset required key: %s", key)
		}
	}

	// Check if key exists
	if !viper.IsSet(key) {
		return fmt.Errorf("key '%s' does not exist", key)
	}

	// Confirmation prompt
	if !opts.Force {
		if !cli.ConfirmAction(fmt.Sprintf("Delete key '%s'?", key)) {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Load config file
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return fmt.Errorf("config file not found")
	}

	// Read JSON file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Delete key (supports dot notation)
	if err := deleteNestedKey(config, key); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	// Write back to JSON file
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate JSON: %w", err)
	}

	if err := os.WriteFile(configFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Removed: %s\n", key)

	return nil
}

func deleteNestedKey(config map[string]any, key string) error {
	parts := strings.Split(key, ".")

	if len(parts) == 1 {
		// Top-level key
		delete(config, key)
		return nil
	}

	// Nested key
	current := config
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, ok := current[part].(map[string]any); ok {
			current = next
		} else {
			return fmt.Errorf("key not found: %s", key)
		}
	}

	delete(current, parts[len(parts)-1])
	return nil
}
