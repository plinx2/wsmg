package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/git"
	"github.com/plinx2/wsmg/internal/workspace"
	"github.com/spf13/cobra"
)

func newWorkspaceCreateCmd() *cobra.Command {
	// Define options
	opts := &struct {
		Repos      []string `flag:"repos" short:"r" usage:"Specify repositories to include (comma-separated, skips interactive mode)"`
		BaseBranch string   `flag:"base-branch" short:"b" default:"" usage:"Base branch for working branches (default: main/master)"`
		NoVSCode   bool     `flag:"no-vscode" default:"false" usage:"Do not create VSCode workspace file"`
	}{}

	// Define command
	cmd := &cobra.Command{
		Use:   "create <ticket-name>",
		Short: "Create a workspace",
		Long: `Create a new workspace.

Interactively select repositories, create working branches for each repository,
and place them in the workspace directory using Git worktree.

The workspace directory structure will mirror the repos directory structure:
  workspaces/<ticket>/provider.com/organization/repository`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ticketName := args[0]
			return runWorkspaceCreate(ticketName, opts.Repos, opts.BaseBranch, opts.NoVSCode)
		},
	}

	// Bind flags (auto-generated from tags)
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runWorkspaceCreate(ticketName string, repoList []string, baseBranch string, noVSCode bool) error {
	reposDir := config.GetReposDir()
	workspacesDir := config.GetWorkspacesDir()

	// Check if workspace already exists
	if workspace.WorkspaceExists(workspacesDir, ticketName) {
		return fmt.Errorf("workspace '%s' already exists", ticketName)
	}

	// Check if repos directory exists
	if _, err := os.Stat(reposDir); os.IsNotExist(err) {
		return fmt.Errorf("repos directory does not exist: %s", reposDir)
	}

	// Find repositories
	fmt.Fprintf(os.Stderr, "Scanning repositories in %s...\n", reposDir)
	allRepos, err := git.FindRepositories(reposDir)
	if err != nil {
		return fmt.Errorf("failed to find repositories: %w", err)
	}

	if len(allRepos) == 0 {
		return fmt.Errorf("no repositories found")
	}

	// Determine repositories to use
	var selectedRepos []*git.Repository
	if len(repoList) > 0 {
		// Use repositories specified via command line
		selectedRepos = selectReposByNames(allRepos, repoList, reposDir)
		if len(selectedRepos) == 0 {
			return fmt.Errorf("specified repositories not found")
		}
	} else {
		// Interactive selection
		selectedRepos, err = selectReposInteractive(allRepos, reposDir)
		if err != nil {
			return err
		}
		if len(selectedRepos) == 0 {
			fmt.Println("No repositories selected.")
			return nil
		}
	}

	fmt.Printf("\nSelected repositories: %d\n", len(selectedRepos))
	for _, repo := range selectedRepos {
		fmt.Printf("  - %s\n", repo.RelativePath(reposDir))
	}

	// Confirmation
	if !confirmAction("\nCreate workspace?") {
		fmt.Println("Cancelled.")
		return nil
	}

	// Create workspace directory
	workspacePath, err := workspace.CreateWorkspaceDir(workspacesDir, ticketName)
	if err != nil {
		return err
	}

	fmt.Printf("\nCreating workspace: %s\n\n", workspacePath)

	// Get copy patterns from config
	copyPatterns := config.GetCopyPatterns()

	// Create worktrees for each repository
	var workspaceRepos []workspace.RepositoryInfo
	successCount := 0
	errorCount := 0

	for _, repo := range selectedRepos {
		relPath := repo.RelativePath(reposDir)
		fmt.Printf("Processing %s...", relPath)

		// Build repository path within workspace, maintaining provider/organization structure
		// Example: workspaces/TICKET-001/github.com/org/repo
		worktreePath := filepath.Join(workspacePath, relPath)

		// Determine base branch
		base := baseBranch
		if base == "" {
			// Get default branch
			defaultBranch, err := git.GetDefaultBranch(repo.Path)
			if err != nil {
				// Fallback to current branch
				base = repo.Branch
			} else {
				base = defaultBranch
			}
		}

		// Create worktree
		if err := git.AddWorktree(repo.Path, worktreePath, ticketName, base); err != nil {
			fmt.Printf(" ✗ failed: %v\n", err)
			errorCount++
			continue
		}

		fmt.Println(" ✓")
		successCount++

		// Copy environment files from original repository to worktree
		copiedFiles, err := workspace.CopyEnvironmentFiles(repo.Path, worktreePath, copyPatterns)
		if err == nil && len(copiedFiles) > 0 {
			fmt.Printf("  Copied: %s\n", strings.Join(copiedFiles, ", "))
		}

		// Add to workspace info
		workspaceRepos = append(workspaceRepos, workspace.RepositoryInfo{
			Name:         repo.Name,
			RelativePath: relPath,
			Branch:       ticketName,
		})
	}

	// Generate VSCode workspace file
	if !noVSCode && len(workspaceRepos) > 0 {
		fmt.Print("\nGenerating VSCode workspace file...")
		if err := workspace.GenerateVSCodeWorkspace(workspacePath, ticketName, workspaceRepos); err != nil {
			fmt.Printf(" ✗ failed: %v\n", err)
		} else {
			fmt.Println(" ✓")
		}
	}

	// Display summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Success: %d repositories\n", successCount)
	if errorCount > 0 {
		fmt.Printf("  Error:   %d repositories\n", errorCount)
	}
	fmt.Printf("\nWorkspace created: %s\n", workspacePath)

	if !noVSCode {
		fmt.Printf("\nTo open in VSCode:\n")
		fmt.Printf("  code %s/%s.code-workspace\n", workspacePath, ticketName)
	}

	return nil
}

func selectReposByNames(allRepos []*git.Repository, names []string, baseDir string) []*git.Repository {
	var selected []*git.Repository

	for _, repo := range allRepos {
		relPath := repo.RelativePath(baseDir)
		for _, name := range names {
			if strings.Contains(relPath, name) || strings.Contains(repo.Name, name) {
				selected = append(selected, repo)
				break
			}
		}
	}

	return selected
}

func selectReposInteractive(allRepos []*git.Repository, baseDir string) ([]*git.Repository, error) {
	fmt.Println("\nSelect repositories (enter numbers separated by commas, or 'all' for all):")
	fmt.Println()

	// Display repository list
	for i, repo := range allRepos {
		fmt.Printf("  [%d] %s (branch: %s)\n", i+1, repo.RelativePath(baseDir), repo.Branch)
	}

	fmt.Print("\nSelection (e.g., 1,3,5 or all): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)

	if input == "all" {
		return allRepos, nil
	}

	// Parse numbers
	indices := parseIndices(input)
	var selected []*git.Repository

	for _, idx := range indices {
		if idx > 0 && idx <= len(allRepos) {
			selected = append(selected, allRepos[idx-1])
		}
	}

	return selected, nil
}

func parseIndices(input string) []int {
	var indices []int
	parts := strings.Split(input, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err == nil {
			indices = append(indices, idx)
		}
	}

	return indices
}

func confirmAction(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
