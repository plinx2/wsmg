package main

import (
	"fmt"
	"strings"

	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/repo"
)

// workspaceNotFoundError creates a helpful error message when workspace is not found
func workspaceNotFoundError(workspaceName string, originalErr error) error {
	// Get available workspaces
	workspaces, err := workspaceClient.ListWorkspaces()
	if err != nil || len(workspaces) == 0 {
		return fmt.Errorf("workspace '%s' not found\n\nRun 'wsmg workspace list' to see available workspaces\nRun 'wsmg workspace create %s' to create it",
			workspaceName, workspaceName)
	}

	// Show available workspaces (max 5)
	var names []string
	for i, ws := range workspaces {
		if i >= 5 {
			names = append(names, "...")
			break
		}
		names = append(names, ws.Name)
	}

	return fmt.Errorf("workspace '%s' not found\n\nAvailable workspaces (%d):\n  %s\n\nRun 'wsmg workspace list' for full list\nRun 'wsmg workspace create %s' to create it",
		workspaceName, len(workspaces), strings.Join(names, "\n  "), workspaceName)
}

// repositoryNotFoundError creates a helpful error message when repository is not found
func repositoryNotFoundError(repoName string) error {
	// Get available repositories (limited to avoid performance issues)
	repos, err := repoClient.List(repo.ListInput{})
	if err != nil || len(repos) == 0 {
		return fmt.Errorf("repository '%s' not found\n\nRun 'wsmg repos list' to see available repositories", repoName)
	}

	// Show available repositories (max 10)
	reposDir := config.GetReposDir()
	var names []string
	for i, r := range repos {
		if i >= 10 {
			names = append(names, "...")
			break
		}
		names = append(names, r.RelativePath(reposDir))
	}

	return fmt.Errorf("repository '%s' not found\n\nAvailable repositories (%d):\n  %s\n\nRun 'wsmg repos list' for full list",
		repoName, len(repos), strings.Join(names, "\n  "))
}

// configNotFoundError creates a helpful error message when config is not found
func configNotFoundError() error {
	return fmt.Errorf("configuration not found\n\nRun 'wsmg config init' to create a configuration file\nOr specify config file with: wsmg --config /path/to/config.json")
}

// noRepositoriesError creates a helpful error message when no repositories are found
func noRepositoriesError(reposDir string) error {
	return fmt.Errorf("no repositories found in %s\n\nTo clone repositories:\n  1. Clone manually: git clone <url> %s/<provider>/<owner>/<repo>\n  2. Use remote commands: wsmg remote clone --org <organization>\n\nRun 'wsmg remote --help' for more information",
		reposDir, reposDir)
}
