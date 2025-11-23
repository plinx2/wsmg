package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigValidateOptions represents options for config validate command
type ConfigValidateOptions struct {
	Strict bool `flag:"strict" short:"s" default:"false" usage:"Strict validation (warn about unused keys)"`
}

func newConfigValidateCmd() *cobra.Command {
	opts := &ConfigValidateOptions{}

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long:  `Validate the configuration file for syntax and content errors`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigValidate(c.Context(), opts)
		},
	}

	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigValidate(ctx context.Context, opts *ConfigValidateOptions) error {
	configFile := viper.ConfigFileUsed()

	if configFile == "" {
		return fmt.Errorf("config file not found")
	}

	fmt.Printf("Validating configuration: %s\n\n", configFile)

	// JSON syntax check
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("✗ failed to read config file: %w", err)
	}

	var rawConfig map[string]any
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("✗ JSON syntax error: %w", err)
	}

	// Load configuration
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("✗ failed to load configuration: %w", err)
	}

	errors := []string{}
	warnings := []string{}

	// Check required fields
	if cfg.Repos == "" {
		errors = append(errors, "repos: required field is missing")
	}
	if cfg.Workspaces == "" {
		errors = append(errors, "workspaces: required field is missing")
	}

	// Validate repos directory
	if cfg.Repos != "" {
		if err := validateDirectory(cfg.Repos, "repos"); err != nil {
			errors = append(errors, err.Error())
		} else if warn := checkDirectoryWarnings(cfg.Repos, "repos"); warn != "" {
			warnings = append(warnings, warn)
		}
	}

	// Validate workspaces directory
	if cfg.Workspaces != "" {
		if err := validateDirectory(cfg.Workspaces, "workspaces"); err != nil {
			errors = append(errors, err.Error())
		} else if warn := checkDirectoryWarnings(cfg.Workspaces, "workspaces"); warn != "" {
			warnings = append(warnings, warn)
		}
	}

	// リモート設定のチェック
	for i, remote := range cfg.Remotes {
		if remote.Type == "" {
			errors = append(errors, fmt.Sprintf("remotes.%d.type: required field is missing", i))
		}
		if remote.Host == "" {
			errors = append(errors, fmt.Sprintf("remotes.%d.host: required field is missing", i))
		}
	}

	// 厳格モードの場合、既知のキー以外を警告
	if opts.Strict {
		knownKeys := map[string]bool{
			"repos":      true,
			"workspaces": true,
			"env":        true,
			"remotes":    true,
		}
		for key := range rawConfig {
			if !knownKeys[key] {
				warnings = append(warnings, fmt.Sprintf("unused or unknown key: %s", key))
			}
		}
	}

	// 結果を出力
	if len(errors) > 0 {
		fmt.Println("✗ Configuration file is invalid:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("validation failed")
	}

	fmt.Println("✓ Configuration file is valid")

	if len(warnings) > 0 {
		for _, warn := range warnings {
			fmt.Printf("⚠ Warning: %s\n", warn)
		}
	}

	return nil
}

// validateDirectory validates a directory for existence and permissions
func validateDirectory(path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, try to create it to check if path is valid
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("%s: cannot create directory: %w", name, err)
			}
			// Remove the test directory
			os.Remove(path)
			// Return nil as the path is valid (just doesn't exist yet)
			return nil
		}
		return fmt.Errorf("%s: cannot access directory: %w", name, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s: path exists but is not a directory: %s", name, path)
	}

	// Check if directory is writable
	testFile := filepath.Join(path, ".wsmg_test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("%s: directory is not writable: %w", name, err)
	}
	os.Remove(testFile)

	return nil
}

// checkDirectoryWarnings checks for directory warnings (non-fatal issues)
func checkDirectoryWarnings(path, name string) string {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("%s: directory does not exist: %s (will be created on first use)", name, path)
		}
		return ""
	}

	// Check if directory is empty
	entries, err := os.ReadDir(path)
	if err != nil {
		return ""
	}

	if len(entries) == 0 {
		return fmt.Sprintf("%s: directory is empty: %s", name, path)
	}

	// Check disk space (Unix-like systems)
	// Note: This is a basic check, more sophisticated checks could be added
	if info.Mode().Perm()&0200 == 0 {
		return fmt.Sprintf("%s: directory has restrictive permissions: %s", name, path)
	}

	return ""
}
