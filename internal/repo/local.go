package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/plinx2/wsmg/internal/fs"
	"github.com/plinx2/wsmg/internal/repo/auth"
	"go.yaml.in/yaml/v3"
)

// Local represents a Git repository in the local filesystem
type Local struct {
	// Absolute path to the repository
	path string
	// Git repository
	repo *git.Repository
}

func NewLocal(path string) (*Local, error) {
	r, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, err
	}
	return &Local{path: path, repo: r}, nil
}

func (r *Local) Path() string {
	return r.path
}

func (r *Local) Name() string {
	return filepath.Base(r.path)
}

func (r *Local) DotGitPath() string {
	return filepath.Join(r.path, ".git")
}

func (r *Local) IsOriginal() bool {
	info, err := os.Stat(r.DotGitPath())
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (r *Local) IsWorktree() bool {
	return !r.IsOriginal()
}

func (r *Local) Original() (*Local, error) {
	if r.IsOriginal() {
		return r, nil
	}

	gitPath := r.DotGitPath()
	info, err := os.Stat(gitPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("not a worktree: %s", gitPath)
	}

	gitdir, err := r.gitdir()
	if err != nil {
		return nil, err
	}
	index := strings.Index(gitdir, "/.git/")
	if index == -1 {
		return nil, fmt.Errorf("invalid gitdir: %s", gitdir)
	}
	originalRepoPath := gitdir[:index]
	return NewLocal(originalRepoPath)
}

func (r *Local) Branches() ([]string, error) {
	if r.IsWorktree() {
		original, err := r.Original()
		if err != nil {
			return nil, err
		}
		branches, err := original.Branches()
		if err != nil {
			return nil, err
		}
		return branches, nil
	}
	branches, err := r.repo.Branches()
	if err != nil {
		return nil, err
	}
	defer branches.Close()
	var branchNames []string
	branches.ForEach(func(ref *plumbing.Reference) error {
		branchNames = append(branchNames, ref.Name().Short())
		return nil
	})
	return branchNames, nil
}

func (r *Local) Branch() (string, error) {
	if r.IsWorktree() {
		gitdir, err := r.gitdir()
		if err != nil {
			return "", err
		}
		head, err := os.ReadFile(filepath.Join(gitdir, "HEAD"))
		if err != nil {
			return "", err
		}
		b := strings.TrimSpace(string(head))
		b = strings.TrimPrefix(b, "ref: refs/heads/")
		return b, nil
	}
	head, err := r.repo.Head()
	if err != nil {
		return "", err
	}
	return head.Name().Short(), nil
}

var defaultBranches = []string{"main", "master"}

func (r *Local) DefaultBranch() (string, error) {
	// Get the default branch of remote (origin)
	refs, err := r.repo.References()
	if err != nil {
		return "", err
	}
	defer refs.Close()
	var defaultBranch string
	for {
		ref, err := refs.Next()
		if err != nil {
			break
		}
		// Look for refs/remotes/origin/HEAD which points to the default branch
		if ref.Name().IsRemote() && ref.Name().String() == plumbing.NewRemoteHEADReferenceName(git.DefaultRemoteName).String() {
			target := ref.Target()
			parts := strings.Split(target.String(), "/")
			if len(parts) >= 4 {
				defaultBranch = parts[3]
				break
			}
		}
	}
	if defaultBranch != "" {
		return defaultBranch, nil
	}
	// fallback: try "main" or "master"
	branches, err := r.Branches()
	if err != nil {
		return "", err
	}
	for _, branch := range branches {
		if slices.Contains(defaultBranches, branch) {
			return branch, nil
		}
	}
	return "", fmt.Errorf("default branch not found")
}

func (r *Local) CreateBranch(branch, baseBranch string) (err error) {
	currentBranch, err := r.Branch()
	if err != nil {
		return err
	}

	wt, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	if baseBranch == "" {
		baseBranch = currentBranch
	}

	if currentBranch != baseBranch {
		if err := wt.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(baseBranch),
		}); err != nil {
			return err
		}
	}

	defer func() {
		err = wt.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(currentBranch),
		})
	}()

	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Keep:   true,
		Create: true,
	}); err != nil {
		return err
	}

	return
}

func (r *Local) DeleteBranch(branch string) error {
	// Delete branch reference (refs/heads/<branch>)
	branchRef := plumbing.NewBranchReferenceName(branch)
	storer := r.repo.Storer
	if err := storer.RemoveReference(branchRef); err != nil {
		// If reference doesn't exist, that's okay - continue to delete config
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("failed to delete branch reference: %w", err)
		}
	}

	// Delete branch configuration from .git/config
	// This may fail if the branch config doesn't exist, which is okay
	if err := r.repo.DeleteBranch(branch); err != nil {
		// Check if error message indicates branch not found
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "does not exist") {
			// Branch config doesn't exist, which is fine if we already deleted the reference
			return nil
		}
		return fmt.Errorf("failed to delete branch config: %w", err)
	}

	return nil
}

func (r *Local) CheckoutBranch(branch string) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	return wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
}

func (r *Local) DeleteRemoteBranch(branch string) error {
	if err := r.repo.Push(&git.PushOptions{
		RemoteName: git.DefaultRemoteName,
		Prune:      true,
		FollowTags: true,
		Force:      true,
	}); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil
		}
		return err
	}
	return nil
}

func (r *Local) IsClean() (bool, error) {
	trees, err := r.repo.TreeObjects()
	if err != nil {
		return false, err
	}
	defer trees.Close()
	if _, err := trees.Next(); err != nil {
		if errors.Is(err, io.EOF) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (r *Local) LastCommit() (*Commit, error) {
	head, err := r.repo.Head()
	if err != nil {
		return nil, err
	}
	c, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}
	return &Commit{commit: c}, nil
}

// RelativePath はベースディレクトリからの相対パスを返します
func (r *Local) RelativePath(baseDir string) string {
	relPath, err := filepath.Rel(baseDir, r.path)
	if err != nil {
		return r.path
	}
	return relPath
}

func (r *Local) Fetch(ctx context.Context) error {
	remotes, err := r.repo.Remotes()
	if err != nil {
		return err
	}
	for _, remote := range remotes {
		var authMethod ssh.AuthMethod
		for _, url := range remote.Config().URLs {
			authMethod, err = auth.DetectSSHAuthMethod(url)
			if err != nil {
				continue
			}
			ssh.DefaultAuthBuilder = func(user string) (ssh.AuthMethod, error) {
				return authMethod, nil
			}
			break
		}
		remoteName := remote.Config().Name
		if err := r.repo.FetchContext(ctx, &git.FetchOptions{
			RemoteName: remoteName,
			Prune:      true,
			Auth:       authMethod,
		}); err != nil {
			if errors.Is(err, git.NoErrAlreadyUpToDate) {
				continue
			}
			slog.Error("fetch failed", "error", err)
			return err
		}
	}
	return nil
}

func (r *Local) Pull(ctx context.Context) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	err = wt.PullContext(ctx, &git.PullOptions{})
	if err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil
		}
		return err
	}
	return nil
}

func (r *Local) ListWorktrees() ([]*Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = r.path
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, &GitCommandError{
			Command: cmd.String(),
			Output:  stderr.String(),
			Err:     err,
		}
	}
	return parseWorktreeList(stdout.String())
}

func (r *Local) AddWorktree(worktreePath, branch, baseBranch string) error {
	if r.IsWorktree() {
		return fmt.Errorf("repository is already a worktree")
	}
	branches, err := r.Branches()
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch {
	case slices.Contains(branches, branch):
		cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
	case baseBranch != "":
		cmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath, baseBranch)
	default:
		cmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath)
	}
	cmd.Dir = r.path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitCommandError{
			Command: cmd.String(),
			Output:  string(output),
			Err:     err,
		}
	}

	return nil
}

func (r *Local) RemoveWorktree(worktreePath string, force bool) error {
	if r.IsWorktree() {
		return fmt.Errorf("repository is a worktree")
	}
	var cmd *exec.Cmd
	switch {
	case force:
		cmd = exec.Command("git", "worktree", "remove", "--force", worktreePath)
	default:
		cmd = exec.Command("git", "worktree", "remove", worktreePath)
	}
	cmd.Dir = r.path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitCommandError{
			Command: cmd.String(),
			Output:  string(output),
			Err:     err,
		}
	}
	return nil
}

func (r *Local) MarshalJSON() ([]byte, error) {
	branch, err := r.Branch()
	if err != nil {
		return nil, err
	}
	defaultBranch, err := r.DefaultBranch()
	if err != nil {
		return nil, err
	}
	lastCommit, err := r.LastCommit()
	if err != nil {
		return nil, err
	}
	isClean, err := r.IsClean()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(map[string]any{
		"path":          r.Path(),
		"name":          r.Name(),
		"branch":        branch,
		"defaultBranch": defaultBranch,
		"lastCommit":    lastCommit.Hash(),
		"lastMessage":   lastCommit.Message(),
		"lastModified":  lastCommit.Timestamp().Format("2006-01-02T15:04:05Z07:00"),
		"isClean":       isClean,
	}, "", "  ")
}

func (r *Local) MarshalYAML() ([]byte, error) {
	branch, err := r.Branch()
	if err != nil {
		return nil, err
	}
	defaultBranch, err := r.DefaultBranch()
	if err != nil {
		return nil, err
	}
	lastCommit, err := r.LastCommit()
	if err != nil {
		return nil, err
	}
	isClean, err := r.IsClean()
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(map[string]any{
		"path":          r.Path(),
		"name":          r.Name(),
		"branch":        branch,
		"defaultBranch": defaultBranch,
		"lastCommit":    lastCommit.Hash(),
		"lastMessage":   lastCommit.Message(),
		"lastModified":  lastCommit.Timestamp().Format("2006-01-02T15:04:05Z07:00"),
		"isClean":       isClean,
	})
}

type Client struct {
	reposDir string
}

func NewClient(reposDir string) (*Client, error) {
	if _, err := os.Stat(reposDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("repositories directory does not exist: %w", err)
		}
		return nil, fmt.Errorf("failed to stat repositories directory: %w", err)
	}

	return &Client{
		reposDir: reposDir,
	}, nil
}

type ListSortField string

const (
	ListSortFieldPath       ListSortField = "path"
	ListSortFieldName       ListSortField = "name"
	ListSortFieldLastCommit ListSortField = "lastcommit"
)

type ListInput struct {
	Filter string
	Sort   ListSortField
}

// List lists all repositories in the repositories directory
func (c *Client) List(input ListInput) ([]*Local, error) {
	paths, err := fs.Find(c.reposDir, func(path string) (bool, error) {
		if filepath.Base(path) != ".git" {
			return false, nil
		}
		if input.Filter != "" && !strings.Contains(path, input.Filter) {
			return false, nil
		}
		return true, fs.ErrStop
	})
	if err != nil {
		return nil, err
	}

	var repos []*Local
	for _, path := range paths {
		repoPath := filepath.Dir(path)
		r, err := NewLocal(repoPath)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}

	switch input.Sort {
	case ListSortFieldPath:
		sort.Slice(repos, func(i, j int) bool {
			return repos[i].Path() < repos[j].Path()
		})
	case ListSortFieldName:
		sort.Slice(repos, func(i, j int) bool {
			return repos[i].Name() < repos[j].Name()
		})
	case ListSortFieldLastCommit:
		sort.Slice(repos, func(i, j int) bool {
			commitI, err := repos[i].LastCommit()
			if err != nil {
				return false
			}
			commitJ, err := repos[j].LastCommit()
			if err != nil {
				return false
			}
			return commitI.Timestamp().After(commitJ.Timestamp())
		})
	}

	return repos, nil
}

// Sync synchronizes repositories by fetching, checking out default branch, and pulling
func (c *Client) Sync(ctx context.Context, filter string, prune bool) error {
	repos, err := c.List(ListInput{
		Filter: filter,
		Sort:   ListSortFieldPath,
	})
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories found")
	}

	// Parallel synchronization
	type syncResult struct {
		repoName string
		err      error
	}

	results := make(chan syncResult, len(repos))
	var wg sync.WaitGroup

	// Limit concurrent operations to avoid overwhelming the system
	semaphore := make(chan struct{}, 10) // Max 10 concurrent sync operations

	for _, repo := range repos {
		wg.Add(1)
		go func(r *Local) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Fetch
			if err := r.Fetch(ctx); err != nil {
				results <- syncResult{repoName: r.Name(), err: fmt.Errorf("fetch failed: %w", err)}
				return
			}

			// Get default branch
			defaultBranch, err := r.DefaultBranch()
			if err != nil {
				results <- syncResult{repoName: r.Name(), err: fmt.Errorf("failed to get default branch: %w", err)}
				return
			}

			// Checkout default branch if not already on it
			currentBranch, err := r.Branch()
			if err != nil {
				results <- syncResult{repoName: r.Name(), err: fmt.Errorf("failed to get current branch: %w", err)}
				return
			}

			if currentBranch != defaultBranch {
				if err := r.CheckoutBranch(defaultBranch); err != nil {
					results <- syncResult{repoName: r.Name(), err: fmt.Errorf("checkout failed: %w", err)}
					return
				}
			}

			// Pull
			if err := r.Pull(ctx); err != nil {
				results <- syncResult{repoName: r.Name(), err: fmt.Errorf("pull failed: %w", err)}
				return
			}

			results <- syncResult{repoName: r.Name(), err: nil}
		}(repo)
	}

	// Close results channel when all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect errors
	var syncErrors []error
	for res := range results {
		if res.err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("%s: %w", res.repoName, res.err))
		}
	}

	if len(syncErrors) > 0 {
		return fmt.Errorf("%d repositories failed to sync", len(syncErrors))
	}

	return nil
}

// GetModifiedFiles returns a list of modified files (Changes)
func (l *Local) GetModifiedFiles() ([]string, error) {
	wt, err := l.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var modifiedFiles []string
	for file, stat := range status {
		// Include modified, added, deleted, renamed, copied files
		// Exclude untracked files (handled separately)
		if stat.Staging != git.Untracked && stat.Worktree != git.Untracked {
			if stat.Staging != git.Unmodified || stat.Worktree != git.Unmodified {
				modifiedFiles = append(modifiedFiles, file)
			}
		}
	}

	return modifiedFiles, nil
}

// GetUntrackedFiles returns a list of untracked files
func (l *Local) GetUntrackedFiles() ([]string, error) {
	wt, err := l.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var untrackedFiles []string
	for file, stat := range status {
		if stat.Worktree == git.Untracked {
			untrackedFiles = append(untrackedFiles, file)
		}
	}

	return untrackedFiles, nil
}

// UncommittedFiles returns a list of uncommitted files (modified and untracked)
func (l *Local) UncommittedFiles() ([]string, error) {
	// Get modified files
	modifiedFiles, err := l.GetModifiedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get modified files: %w", err)
	}

	// Get untracked files
	untrackedFiles, err := l.GetUntrackedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get untracked files: %w", err)
	}

	// Combine all files
	allFiles := append(modifiedFiles, untrackedFiles...)
	return allFiles, nil
}

func (l *Local) gitdir() (string, error) {
	if !l.IsWorktree() {
		return "", fmt.Errorf("not a worktree")
	}
	gitdir, err := os.ReadFile(l.DotGitPath())
	if err != nil {
		return "", err
	}
	contents := strings.TrimPrefix(string(gitdir), "gitdir: ")
	return strings.TrimSpace(contents), nil
}
