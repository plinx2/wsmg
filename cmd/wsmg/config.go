package main

import "github.com/spf13/cobra"

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Manage configuration files: display, edit, and validate.`,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(newConfigInitCmd())
	configCmd.AddCommand(newConfigShowCmd())
	configCmd.AddCommand(newConfigGetCmd())
	configCmd.AddCommand(newConfigSetCmd())
	configCmd.AddCommand(newConfigUnsetCmd())
	configCmd.AddCommand(newConfigEditCmd())
	configCmd.AddCommand(newConfigPathCmd())
	configCmd.AddCommand(newConfigValidateCmd())
}
