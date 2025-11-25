package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/plinx2/wsmg/internal/repo"
)

// Workspace represents a workspace
type Workspace struct {
	Name         string        // Workspace name (ticket name)
	Path         string        // Workspace path
	Repositories []*repo.Local // Repositories in the workspace
	Created      time.Time     // Creation time
	Modified     time.Time     // Last modified time
}

// vscodeWorkspace represents VSCode workspace configuration
type vscodeWorkspace struct {
	Folders  []vscodeFolder `json:"folders"`
	Settings map[string]any `json:"settings,omitempty"`
}

// vscodeFolder represents a folder in VSCode workspace
type vscodeFolder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// generateVSCodeWorkspace generates or updates VSCode workspace file
func (c *Client) generateVSCodeWorkspace(workspacePath, workspaceName string, repos []*repo.Local) error {
	workspaceFile := filepath.Join(workspacePath, workspaceName+".code-workspace")

	// Load existing workspace file if it exists
	var workspace vscodeWorkspace
	if data, err := os.ReadFile(workspaceFile); err == nil {
		// Existing workspace file found, load it
		if err := json.Unmarshal(data, &workspace); err != nil {
			// If unmarshal fails, start fresh
			workspace = vscodeWorkspace{
				Folders:  make([]vscodeFolder, 0, len(repos)+1),
				Settings: map[string]any{},
			}
		}
	} else {
		// No existing file, create new
		workspace = vscodeWorkspace{
			Folders:  make([]vscodeFolder, 0, len(repos)+1),
			Settings: map[string]any{},
		}
	}

	// Build map of existing folders for deduplication
	existingFolders := make(map[string]vscodeFolder)
	for _, folder := range workspace.Folders {
		existingFolders[folder.Path] = folder
	}

	// Ensure workspace root folder exists (for CURSOR.md, README.md, etc.)
	if _, exists := existingFolders["."]; !exists {
		workspace.Folders = append([]vscodeFolder{{
			Name: workspaceName,
			Path: ".",
		}}, workspace.Folders...)
	}

	// Add or update repository folders
	for _, r := range repos {
		relPath, err := filepath.Rel(workspacePath, r.Path())
		if err != nil {
			continue
		}
		if _, exists := existingFolders[relPath]; !exists {
			workspace.Folders = append(workspace.Folders, vscodeFolder{
				Name: r.Name(),
				Path: relPath,
			})
		}
	}

	// Save workspace file
	data, err := json.MarshalIndent(workspace, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate JSON: %w", err)
	}

	if err := os.WriteFile(workspaceFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write workspace file: %w", err)
	}

	// Update .vscode/settings.json
	if err := c.updateVSCodeSettings(workspacePath, repos); err != nil {
		return fmt.Errorf("failed to update VSCode settings: %w", err)
	}

	return nil
}

// updateVSCodeSettings updates .vscode/settings.json to hide repositories from root folder
func (c *Client) updateVSCodeSettings(workspacePath string, repos []*repo.Local) error {
	vscodeDir := filepath.Join(workspacePath, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vscode directory: %w", err)
	}

	settingsFile := filepath.Join(vscodeDir, "settings.json")

	// Load existing settings if they exist
	var settings map[string]any
	if data, err := os.ReadFile(settingsFile); err == nil {
		// Existing settings found, load them
		if err := json.Unmarshal(data, &settings); err != nil {
			// If unmarshal fails, start fresh
			settings = make(map[string]any)
		}
	} else {
		// No existing file, create new
		settings = make(map[string]any)
	}

	// Get or create files.exclude section
	var filesExclude map[string]any
	if existingExclude, ok := settings["files.exclude"].(map[string]any); ok {
		filesExclude = existingExclude
	} else {
		filesExclude = make(map[string]any)
	}

	// Add repository directories to files.exclude
	// This hides them from the workspace root folder view
	for _, r := range repos {
		relPath, err := filepath.Rel(workspacePath, r.Path())
		if err != nil {
			continue
		}
		filesExclude[relPath] = true
	}

	// Update files.exclude in settings
	settings["files.exclude"] = filesExclude

	// Save settings file with proper formatting
	settingsData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate settings JSON: %w", err)
	}

	if err := os.WriteFile(settingsFile, settingsData, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// copyEnvironmentFiles copies files matching specified patterns from source to target
func (c *Client) copyEnvironmentFiles(sourceRepo, targetWorktree string, patterns []string) ([]string, error) {
	var copiedFiles []string
	copiedPaths := make(map[string]bool)

	err := filepath.Walk(sourceRepo, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if path == sourceRepo {
			return nil
		}

		relPath, err := filepath.Rel(sourceRepo, path)
		if err != nil {
			return nil
		}

		// Limit to 3 levels deep
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > 3 {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Check if matches any pattern
		if !matchesPattern(relPath, patterns) {
			return nil
		}

		if copiedPaths[relPath] {
			return nil
		}

		targetPath := filepath.Join(targetWorktree, relPath)

		if info.IsDir() {
			if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
				return nil
			}
			copiedFiles = append(copiedFiles, relPath+"/")
			copiedPaths[relPath] = true
		} else {
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return nil
			}
			if err := copyFile(path, targetPath); err != nil {
				return nil
			}
			copiedFiles = append(copiedFiles, relPath)
			copiedPaths[relPath] = true
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan files: %w", err)
	}

	return copiedFiles, nil
}

// matchesPattern checks if the relative path matches any of the specified patterns
func matchesPattern(relPath string, patterns []string) bool {
	// Normalize path separator (Windows support)
	relPath = filepath.ToSlash(relPath)

	for _, pattern := range patterns {
		// Normalize pattern
		pattern = filepath.ToSlash(pattern)

		// Directory pattern (e.g., .vscode/*)
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			// .vscode/* matches .vscode/settings.json, etc.
			if relPath == prefix || strings.HasPrefix(relPath, prefix+"/") {
				return true
			}
			continue
		}

		// Wildcard pattern matching
		matched, err := filepath.Match(pattern, relPath)
		if err != nil {
			// On pattern error, try exact match
			if relPath == pattern {
				return true
			}
			continue
		}

		if matched {
			return true
		}

		// Also try matching basename (for backward compatibility)
		basename := filepath.Base(relPath)
		matched, err = filepath.Match(pattern, basename)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// copyFile copies a file from src to dst (private helper)
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Get file info to preserve permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Set permissions
	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return err
	}

	return nil
}

// Client manages workspace operations
type Client struct {
	reposDir      string
	workspacesDir string
	repoClient    *repo.Client
}

// NewClient creates a new workspace client
func NewClient(reposDir, workspacesDir string) (*Client, error) {
	// Check if repos directory exists
	if _, err := os.Stat(reposDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("repositories directory does not exist: %s", reposDir)
		}
		return nil, fmt.Errorf("failed to stat repositories directory: %w", err)
	}

	// Create repo client
	repoClient, err := repo.NewClient(reposDir)
	if err != nil {
		return nil, err
	}

	return &Client{
		reposDir:      reposDir,
		workspacesDir: workspacesDir,
		repoClient:    repoClient,
	}, nil
}

// CreateWorkspaceInput contains input parameters for workspace creation
type CreateWorkspaceInput struct {
	Name            string
	RepoNames       []string
	BaseBranch      string
	CopyPatterns    []string
	NoVSCode        bool
	CopyUncommitted bool
}

// CreateWorkspaceResult contains the result of workspace creation
type CreateWorkspaceResult struct {
	WorkspacePath          string
	SuccessCount           int
	ErrorCount             int
	Repos                  []*repo.Local
	UncommittedFilesCopied int
}

// CreateWorkspace creates a new workspace
func (c *Client) CreateWorkspace(ctx context.Context, input CreateWorkspaceInput) (*CreateWorkspaceResult, error) {
	workspacePath := filepath.Join(c.workspacesDir, input.Name)

	// Check if workspace already exists
	if _, err := os.Stat(workspacePath); err == nil {
		return nil, fmt.Errorf("workspace '%s' already exists", input.Name)
	}

	// Find all repositories
	allRepos, err := c.repoClient.List(repo.ListInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to find repositories: %w", err)
	}

	if len(allRepos) == 0 {
		return nil, fmt.Errorf("no repositories found")
	}

	// Filter repositories by names
	var selectedRepos []*repo.Local
	if len(input.RepoNames) > 0 {
		for _, r := range allRepos {
			relPath := r.RelativePath(c.reposDir)
			for _, name := range input.RepoNames {
				if strings.Contains(relPath, name) || strings.Contains(r.Name(), name) {
					selectedRepos = append(selectedRepos, r)
					break
				}
			}
		}
		if len(selectedRepos) == 0 {
			return nil, fmt.Errorf("specified repositories not found")
		}
	} else {
		selectedRepos = allRepos
	}

	// Create workspace directory
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	result := &CreateWorkspaceResult{
		WorkspacePath: workspacePath,
		Repos:         []*repo.Local{},
	}

	// Create worktrees for each repository
	for _, r := range selectedRepos {
		relPath := r.RelativePath(c.reposDir)
		worktreePath := filepath.Join(workspacePath, relPath)

		currentBranch, err := r.Branch()
		if err != nil {
			return nil, err
		}

		base := currentBranch
		if input.BaseBranch != "" && base != input.BaseBranch {
			base = input.BaseBranch
		}

		if err := r.AddWorktree(worktreePath, input.Name, base); err != nil {
			return nil, err
		}

		result.SuccessCount++

		// Copy environment files
		if len(input.CopyPatterns) > 0 {
			c.copyEnvironmentFiles(r.Path(), worktreePath, input.CopyPatterns)
		}

		// Copy uncommitted changes and untracked files if requested
		if input.CopyUncommitted {
			copiedCount, err := c.copyUncommittedFiles(r, worktreePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to copy uncommitted files for %s: %v\n", r.Name(), err)
			} else if copiedCount > 0 {
				result.UncommittedFilesCopied += copiedCount
			}
		}

		// Add worktree to result
		worktreeRepo, err := repo.NewLocal(worktreePath)
		if err == nil {
			result.Repos = append(result.Repos, worktreeRepo)
		}
	}

	// Generate VSCode workspace file and .vscode/settings.json
	if !input.NoVSCode && len(result.Repos) > 0 {
		if err := c.generateVSCodeWorkspace(workspacePath, input.Name, result.Repos); err != nil {
			// VSCode workspace file generation error is not critical
			fmt.Fprintf(os.Stderr, "Warning: failed to generate VSCode workspace file: %v\n", err)
		}
	}

	return result, nil
}

// AddRepositoriesToWorkspace adds repositories to an existing workspace
func (c *Client) AddRepositoriesToWorkspace(workspaceName string, repoNames []string, copyUncommitted bool) (int, error) {
	workspacePath := filepath.Join(c.workspacesDir, workspaceName)

	// Check if workspace exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return 0, fmt.Errorf("workspace '%s' does not exist", workspaceName)
	}

	// Get existing workspace information
	ws, err := c.GetWorkspace(workspaceName)
	if err != nil {
		return 0, fmt.Errorf("failed to get workspace info: %w", err)
	}

	// Find all repositories
	allRepos, err := c.repoClient.List(repo.ListInput{})
	if err != nil {
		return 0, fmt.Errorf("failed to find repositories: %w", err)
	}

	if len(allRepos) == 0 {
		return 0, fmt.Errorf("no repositories found")
	}

	// Filter repositories by names
	var selectedRepos []*repo.Local
	existingPaths := make(map[string]bool)
	for _, r := range ws.Repositories {
		existingPaths[r.Path()] = true
	}

	for _, r := range allRepos {
		// Skip if repository already exists in workspace
		relPath := r.RelativePath(c.reposDir)
		worktreePath := filepath.Join(workspacePath, relPath)
		if existingPaths[worktreePath] {
			continue
		}

		// Check if repository name matches
		for _, name := range repoNames {
			if strings.Contains(relPath, name) || strings.Contains(r.Name(), name) {
				selectedRepos = append(selectedRepos, r)
				break
			}
		}
	}

	if len(selectedRepos) == 0 {
		return 0, fmt.Errorf("no new repositories to add")
	}

	// Determine the branch name from existing repositories
	var branchName string
	if len(ws.Repositories) > 0 {
		// Use the branch name from the first existing repository
		branchName, err = ws.Repositories[0].Branch()
		if err != nil {
			// Fallback to workspace name
			branchName = workspaceName
		}
	} else {
		branchName = workspaceName
	}

	// Create worktrees for new repositories
	var addedRepos []*repo.Local
	uncommittedFilesCopied := 0
	for _, r := range selectedRepos {
		relPath := r.RelativePath(c.reposDir)
		worktreePath := filepath.Join(workspacePath, relPath)

		// Determine base branch
		base, err := r.DefaultBranch()
		if err != nil {
			currentBranch, err := r.Branch()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get branch for %s: %v\n", r.Name(), err)
				continue
			}
			base = currentBranch
		}

		// Create worktree
		if err := r.AddWorktree(worktreePath, branchName, base); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create worktree for %s: %v\n", r.Name(), err)
			continue
		}

		// Copy uncommitted changes and untracked files if requested
		if copyUncommitted {
			copiedCount, err := c.copyUncommittedFiles(r, worktreePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to copy uncommitted files for %s: %v\n", r.Name(), err)
			} else if copiedCount > 0 {
				uncommittedFilesCopied += copiedCount
			}
		}

		// Add worktree to result
		worktreeRepo, err := repo.NewLocal(worktreePath)
		if err == nil {
			addedRepos = append(addedRepos, worktreeRepo)
		}
	}

	if len(addedRepos) == 0 {
		return 0, fmt.Errorf("failed to add any repositories")
	}

	// Update VSCode workspace file and .vscode/settings.json
	// This will merge with existing configuration
	allReposInWorkspace := append(ws.Repositories, addedRepos...)
	if err := c.generateVSCodeWorkspace(workspacePath, workspaceName, allReposInWorkspace); err != nil {
		// VSCode workspace file generation error is not critical
		fmt.Fprintf(os.Stderr, "Warning: failed to update VSCode workspace file: %v\n", err)
	}

	return uncommittedFilesCopied, nil
}

// RemoveRepositoriesFromWorkspace removes repositories from an existing workspace
func (c *Client) RemoveRepositoriesFromWorkspace(workspaceName string, repoRelPaths []string, force bool) (int, error) {
	workspacePath := filepath.Join(c.workspacesDir, workspaceName)

	// Check if workspace exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return 0, fmt.Errorf("workspace '%s' does not exist", workspaceName)
	}

	// Get existing workspace information
	ws, err := c.GetWorkspace(workspaceName)
	if err != nil {
		return 0, fmt.Errorf("failed to get workspace info: %w", err)
	}

	if len(ws.Repositories) == 0 {
		return 0, fmt.Errorf("workspace has no repositories")
	}

	// Build map of repositories to remove
	toRemove := make(map[string]bool)
	for _, relPath := range repoRelPaths {
		toRemove[relPath] = true
	}

	// Find and remove repositories
	var remainingRepos []*repo.Local
	removedCount := 0

	for _, r := range ws.Repositories {
		relPath := r.RelativePath(c.reposDir)
		absPath := r.Path()

		if toRemove[relPath] || toRemove[absPath] {
			// Remove worktree
			original, err := r.Original()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get original repository for %s: %v\n", r.Name(), err)
				continue
			}

			if err := original.RemoveWorktree(r.Path(), force); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree for %s: %v\n", r.Name(), err)
				continue
			}

			// Remove directory
			if err := os.RemoveAll(r.Path()); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove directory %s: %v\n", r.Path(), err)
				continue
			}

			removedCount++
		} else {
			// Repository should remain
			remainingRepos = append(remainingRepos, r)
		}
	}

	if removedCount == 0 {
		return 0, fmt.Errorf("no matching repositories found to remove")
	}

	// Update VSCode workspace file and .vscode/settings.json
	if err := c.generateVSCodeWorkspace(workspacePath, workspaceName, remainingRepos); err != nil {
		// VSCode workspace file generation error is not critical
		fmt.Fprintf(os.Stderr, "Warning: failed to update VSCode workspace file: %v\n", err)
	}

	return removedCount, nil
}

// RenameWorkspace renames a workspace, including directory, branches, and VSCode files
func (c *Client) RenameWorkspace(oldName, newName string) error {
	oldPath := filepath.Join(c.workspacesDir, oldName)
	newPath := filepath.Join(c.workspacesDir, newName)

	// Check if old workspace exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("workspace '%s' does not exist", oldName)
	}

	// Check if new workspace already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("workspace '%s' already exists", newName)
	}

	// Get workspace info
	ws, err := c.GetWorkspace(oldName)
	if err != nil {
		return fmt.Errorf("failed to get workspace info: %w", err)
	}

	// Rename branches in all repositories
	for _, r := range ws.Repositories {
		currentBranch, err := r.Branch()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get branch for %s: %v\n", r.Name(), err)
			continue
		}

		// Only rename if the branch name matches the old workspace name
		if currentBranch == oldName {
			// Create new branch from current branch
			if err := r.CreateBranch(newName, currentBranch); err != nil {
				return fmt.Errorf("failed to create new branch '%s' in %s: %w", newName, r.Name(), err)
			}

			// Checkout new branch
			if err := r.CheckoutBranch(newName); err != nil {
				return fmt.Errorf("failed to checkout new branch in %s: %w", r.Name(), err)
			}

			// Delete old branch
			if err := r.DeleteBranch(currentBranch); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete old branch '%s' in %s: %v\n", currentBranch, r.Name(), err)
			}
		}
	}

	// Rename workspace directory
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename workspace directory: %w", err)
	}

	// Get updated repository list with new paths
	updatedRepos := make([]*repo.Local, 0, len(ws.Repositories))
	for _, r := range ws.Repositories {
		// Calculate new path
		relPath := r.RelativePath(oldPath)
		newRepoPath := filepath.Join(newPath, relPath)

		// Create new Local instance with updated path
		updatedRepo, err := repo.NewLocal(newRepoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load repository at %s: %v\n", newRepoPath, err)
			continue
		}
		updatedRepos = append(updatedRepos, updatedRepo)
	}

	// Remove old VSCode workspace file
	oldWorkspaceFile := filepath.Join(newPath, oldName+".code-workspace")
	if err := os.Remove(oldWorkspaceFile); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove old workspace file: %v\n", err)
	}

	// Generate new VSCode workspace file
	if err := c.generateVSCodeWorkspace(newPath, newName, updatedRepos); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to generate new VSCode workspace file: %v\n", err)
	}

	return nil
}

// ListWorkspaces lists all workspaces
func (c *Client) ListWorkspaces() ([]*Workspace, error) {
	entries, err := os.ReadDir(c.workspacesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Workspace{}, nil
		}
		return nil, err
	}

	// Filter directory entries
	var dirNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirNames = append(dirNames, entry.Name())
		}
	}

	if len(dirNames) == 0 {
		return []*Workspace{}, nil
	}

	var workspaces []*Workspace
	for _, name := range dirNames {
		ws, err := c.GetWorkspace(name)
		if err != nil {
			continue
		}
		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

// GetWorkspace gets workspace information by name
func (c *Client) GetWorkspace(name string) (*Workspace, error) {
	workspacePath := filepath.Join(c.workspacesDir, name)
	info, err := os.Stat(workspacePath)
	if err != nil {
		return nil, err
	}

	ws := &Workspace{
		Name:     name,
		Path:     workspacePath,
		Created:  info.ModTime(),
		Modified: info.ModTime(),
	}

	// Find repositories in workspace
	// Create repo.Client with workspace path and call List()
	wsRepoClient, err := repo.NewClient(workspacePath)
	if err == nil {
		repos, err := wsRepoClient.List(repo.ListInput{})
		if err == nil {
			ws.Repositories = repos
		}
	}

	return ws, nil
}

// DeleteWorkspaceInput contains input parameters for workspace deletion
type DeleteWorkspaceInput struct {
	Name         string
	Force        bool
	KeepBranches bool
	DeleteRemote bool
}

// DeleteWorkspaceResult contains the result of workspace deletion
type DeleteWorkspaceResult struct {
	ProcessedCount int
}

// DeleteWorkspace deletes a workspace
func (c *Client) DeleteWorkspace(input DeleteWorkspaceInput) (*DeleteWorkspaceResult, error) {
	// Get workspace info
	ws, err := c.GetWorkspace(input.Name)
	if err != nil {
		return nil, fmt.Errorf("workspace '%s' not found: %w", input.Name, err)
	}

	result := &DeleteWorkspaceResult{}

	// Remove worktrees and branches for each repository
	for _, worktreeRepo := range ws.Repositories {
		// Get original repository
		originalRepo, err := worktreeRepo.Original()
		if err != nil {
			return nil, err
		}

		// Get branch name
		branch, err := worktreeRepo.Branch()
		if err != nil {
			return nil, err
		}

		// Remove worktree
		if err := originalRepo.RemoveWorktree(worktreeRepo.Path(), input.Force); err != nil {
			return nil, err
		}

		// Delete branch (optional)
		if !input.KeepBranches {
			fmt.Println("Deleting branch", branch)
			if err := originalRepo.DeleteBranch(branch); err != nil {
				fmt.Println("Failed to delete branch", branch, err)
				// Branch deletion failure is not critical
			}
		}

		result.ProcessedCount++
	}

	// Delete workspace directory
	if err := os.RemoveAll(ws.Path); err != nil {
		return result, fmt.Errorf("failed to delete workspace directory: %w", err)
	}

	return result, nil
}

// copyUncommittedFiles copies uncommitted files from original repository to worktree
func (c *Client) copyUncommittedFiles(originalRepo *repo.Local, worktreePath string) (int, error) {
	// Get uncommitted files from original repository
	files, err := originalRepo.UncommittedFiles()
	if err != nil {
		return 0, fmt.Errorf("failed to get uncommitted files: %w", err)
	}

	if len(files) == 0 {
		return 0, nil
	}

	// Copy each file
	copiedCount := 0
	for _, file := range files {
		srcPath := filepath.Join(originalRepo.Path(), file)
		dstPath := filepath.Join(worktreePath, file)

		// Check if source file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			// File might be deleted, skip
			continue
		}

		// Create destination directory
		dstDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return copiedCount, fmt.Errorf("failed to create directory %s: %w", dstDir, err)
		}

		// Copy file
		srcData, err := os.ReadFile(srcPath)
		if err != nil {
			return copiedCount, fmt.Errorf("failed to read file %s: %w", srcPath, err)
		}

		if err := os.WriteFile(dstPath, srcData, 0644); err != nil {
			return copiedCount, fmt.Errorf("failed to write file %s: %w", dstPath, err)
		}

		copiedCount++
	}

	return copiedCount, nil
}

// SyncRepositories synchronizes repositories
func (c *Client) SyncRepositories(ctx context.Context, filter string, prune bool) error {
	return c.repoClient.Sync(ctx, filter, prune)
}
