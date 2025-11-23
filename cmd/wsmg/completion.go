package main

import (
	"fmt"
	"os"

	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/repo"
	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for wsmg.

To load completions:

Bash:
  $ source <(wsmg completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ wsmg completion bash > /etc/bash_completion.d/wsmg
  # macOS:
  $ wsmg completion bash > $(brew --prefix)/etc/bash_completion.d/wsmg

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ wsmg completion zsh > "${fpath[1]}/_wsmg"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ wsmg completion fish | source

  # To load completions for each session, execute once:
  $ wsmg completion fish > ~/.config/fish/completions/wsmg.fish

PowerShell:
  PS> wsmg completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> wsmg completion powershell > wsmg.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell type: %s", args[0])
			}
		},
	}

	return cmd
}

// registerCompletions registers dynamic completion functions for various flags
func registerCompletions() {
	// Workspace name completion
	workspaceNameCompletion := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if workspaceClient == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		workspaces, err := workspaceClient.ListWorkspaces()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var names []string
		for _, ws := range workspaces {
			names = append(names, ws.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}

	// Repository path completion
	repoPathCompletion := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if repoClient == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		repos, err := repoClient.List(repo.ListInput{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		reposDir := config.GetReposDir()
		var paths []string
		for _, r := range repos {
			paths = append(paths, r.RelativePath(reposDir))
		}
		return paths, cobra.ShellCompDirectiveNoFileComp
	}

	// Register for workspace commands
	for _, cmdName := range []string{"show", "delete", "rename", "add", "remove"} {
		if cmd, _, err := rootCmd.Find([]string{"workspace", cmdName}); err == nil {
			cmd.ValidArgsFunction = workspaceNameCompletion
		}
	}

	// Register for repo flag in workspace create/add
	for _, cmdName := range []string{"create", "add"} {
		if cmd, _, err := rootCmd.Find([]string{"workspace", cmdName}); err == nil {
			if flag := cmd.Flags().Lookup("repo"); flag != nil {
				cmd.RegisterFlagCompletionFunc("repo", repoPathCompletion)
			}
		}
	}

	// Format completion for list/show commands
	formatCompletion := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
	}

	// Register format flag completion
	for _, cmdPath := range [][]string{
		{"workspace", "list"},
		{"workspace", "show"},
		{"repos", "list"},
		{"remote", "list"},
	} {
		if cmd, _, err := rootCmd.Find(cmdPath); err == nil {
			if flag := cmd.Flags().Lookup("format"); flag != nil {
				cmd.RegisterFlagCompletionFunc("format", formatCompletion)
			}
		}
	}

	// Provider completion for remote commands
	providerCompletion := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"github", "gitlab"}, cobra.ShellCompDirectiveNoFileComp
	}

	if cmd, _, err := rootCmd.Find([]string{"remote", "list"}); err == nil {
		if flag := cmd.Flags().Lookup("provider"); flag != nil {
			cmd.RegisterFlagCompletionFunc("provider", providerCompletion)
		}
	}
}

func init() {
	rootCmd.AddCommand(newCompletionCmd())
}
