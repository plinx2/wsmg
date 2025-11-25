package main

import (
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/repo"
	"github.com/spf13/cobra"
)

// workspaceNameCompletion returns workspace names for completion
func workspaceNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

// repoPathCompletion returns repository paths for completion
func repoPathCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

// formatCompletion returns format options for completion
func formatCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
}

// providerCompletion returns provider options for completion
func providerCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"github", "gitlab"}, cobra.ShellCompDirectiveNoFileComp
}
