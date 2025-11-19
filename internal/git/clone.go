package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CloneOptions represents options for cloning a repository
type CloneOptions struct {
	URL           string   // Clone URL
	DestPath      string   // Destination path
	Branch        string   // Branch to checkout (optional)
	Depth         int      // Shallow clone depth (0 = full clone)
	SingleBranch  bool     // Clone only the specified branch
	NoCheckout    bool     // Don't checkout HEAD
	Verbose       bool     // Verbose output
}

// Clone clones a Git repository
func Clone(opts CloneOptions) error {
	// Build git clone command
	args := []string{"clone"}

	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}

	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}

	if opts.SingleBranch {
		args = append(args, "--single-branch")
	}

	if opts.NoCheckout {
		args = append(args, "--no-checkout")
	}

	if opts.Verbose {
		args = append(args, "--verbose")
	}

	args = append(args, opts.URL, opts.DestPath)

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(opts.DestPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Execute git clone
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

// IsGitRepository checks if a path is a Git repository
func IsGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir() || !info.IsDir() // Can be directory or file (for worktrees)
}
