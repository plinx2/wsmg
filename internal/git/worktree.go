package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree はGit worktreeを表します
type Worktree struct {
	Path   string // worktreeのパス
	Branch string // ブランチ名
	Commit string // コミットハッシュ
}

// ListWorktrees はリポジトリのworktree一覧を取得します
func ListWorktrees(repoPath string) ([]*Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, &GitCommandError{
			Command: "git worktree list",
			Output:  string(output),
			Err:     err,
		}
	}

	return parseWorktreeList(string(output))
}

// parseWorktreeList はgit worktree listの出力をパースします
func parseWorktreeList(output string) ([]*Worktree, error) {
	var worktrees []*Worktree
	var current *Worktree

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, current)
				current = nil
			}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "worktree":
			current = &Worktree{Path: value}
		case "branch":
			if current != nil {
				// "refs/heads/main" -> "main"
				branch := strings.TrimPrefix(value, "refs/heads/")
				current.Branch = branch
			}
		case "HEAD":
			if current != nil {
				current.Commit = value
			}
		}
	}

	// 最後のエントリを追加
	if current != nil {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// AddWorktree creates a new worktree
// If baseBranch is empty, creates from current HEAD
func AddWorktree(repoPath, worktreePath, branch, baseBranch string) error {
	// Check if branch exists
	branchExists, err := BranchExists(repoPath, branch)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if branchExists {
		// Create worktree from existing branch
		cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
	} else {
		// Create new branch and worktree
		if baseBranch != "" {
			// Create from specified base branch
			cmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath, baseBranch)
		} else {
			// Create from current HEAD
			cmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath)
		}
	}

	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitCommandError{
			Command: fmt.Sprintf("git worktree add %s %s", worktreePath, branch),
			Output:  string(output),
			Err:     err,
		}
	}

	return nil
}

// RemoveWorktree はworktreeを削除します
func RemoveWorktree(repoPath, worktreePath string) error {
	// まず、worktreeのパスを取得
	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "worktree", "remove", absWorktreePath)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		// force で再試行
		cmd = exec.Command("git", "worktree", "remove", "--force", absWorktreePath)
		cmd.Dir = repoPath
		output, err = cmd.CombinedOutput()
		if err != nil {
			return &GitCommandError{
				Command: fmt.Sprintf("git worktree remove %s", absWorktreePath),
				Output:  string(output),
				Err:     err,
			}
		}
	}

	return nil
}

// BranchExists はブランチが存在するかチェックします
func BranchExists(repoPath, branch string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = repoPath
	err := cmd.Run()
	if err != nil {
		// ブランチが存在しない場合はエラーが返る
		return false, nil
	}
	return true, nil
}

// CreateBranch はブランチを作成します
func CreateBranch(repoPath, branch, baseBranch string) error {
	args := []string{"branch", branch}
	if baseBranch != "" {
		args = append(args, baseBranch)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitCommandError{
			Command: "git " + strings.Join(args, " "),
			Output:  string(output),
			Err:     err,
		}
	}

	return nil
}

// DeleteBranch はブランチを削除します
func DeleteBranch(repoPath, branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}

	cmd := exec.Command("git", "branch", flag, branch)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitCommandError{
			Command: fmt.Sprintf("git branch %s %s", flag, branch),
			Output:  string(output),
			Err:     err,
		}
	}

	return nil
}

// GetWorktreePath はブランチに対応するworktreeのパスを取得します
func GetWorktreePath(repoPath, branch string) (string, error) {
	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		return "", err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return wt.Path, nil
		}
	}

	return "", fmt.Errorf("worktree not found for branch: %s", branch)
}
