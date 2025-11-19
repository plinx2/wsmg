package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Repository はGitリポジトリを表します
type Repository struct {
	Path         string    // リポジトリのパス (絶対パス)
	Name         string    // リポジトリ名
	Provider     string    // プロバイダー名 (例: "github.com", "gitlab.com")
	Organization string    // 組織名
	Branch       string    // 現在のブランチ
	LastCommit   string    // 最終コミットのハッシュ（短縮形）
	LastMessage  string    // 最終コミットメッセージ
	LastModified time.Time // 最終コミット日時
	IsClean      bool      // ワーキングツリーがクリーンか
}

// FindRepositories は指定されたディレクトリ配下のGitリポジトリを検索します
// ディレクトリ構造: rootDir/provider/organization/repository
func FindRepositories(rootDir string) ([]*Repository, error) {
	var repos []*Repository

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// アクセスできないディレクトリはスキップ
			return nil
		}

		// .git ディレクトリを見つけたらリポジトリとして認識
		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			repo, err := GetRepositoryInfo(repoPath)
			if err != nil {
				// エラーは無視してスキップ
				return nil
			}

			// プロバイダーと組織情報を解析 (rootDir/provider/organization/repository)
			relPath, err := filepath.Rel(rootDir, repoPath)
			if err == nil {
				parts := strings.Split(filepath.ToSlash(relPath), "/")
				if len(parts) >= 3 {
					repo.Provider = parts[0]
					repo.Organization = parts[1]
				} else if len(parts) == 2 {
					// 後方互換性: organization/repository 形式の場合
					repo.Organization = parts[0]
				}
			}

			repos = append(repos, repo)

			// .git ディレクトリの中は探索しない
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return repos, nil
}

// GetRepositoryInfo はリポジトリの情報を取得します
func GetRepositoryInfo(repoPath string) (*Repository, error) {
	// リポジトリかどうかをチェック
	if !IsRepository(repoPath) {
		return nil, &NotRepositoryError{Path: repoPath}
	}

	repo := &Repository{
		Path: repoPath,
		Name: filepath.Base(repoPath),
	}

	// 現在のブランチを取得
	branch, err := getCurrentBranch(repoPath)
	if err == nil {
		repo.Branch = branch
	}

	// 最終コミット情報を取得
	commitHash, commitMsg, commitTime, err := getLastCommitInfo(repoPath)
	if err == nil {
		repo.LastCommit = commitHash
		repo.LastMessage = commitMsg
		repo.LastModified = commitTime
	}

	// ワーキングツリーの状態を確認
	isClean, err := isWorkingTreeClean(repoPath)
	if err == nil {
		repo.IsClean = isClean
	}

	return repo, nil
}

// IsRepository は指定されたパスがGitリポジトリかどうかを判定します
func IsRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// getCurrentBranch は現在のブランチ名を取得します
func getCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getLastCommitInfo は最終コミットの情報を取得します
func getLastCommitInfo(repoPath string) (hash, message string, timestamp time.Time, err error) {
	// コミットハッシュ（短縮形）
	cmd := exec.Command("git", "log", "-1", "--format=%h")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", "", time.Time{}, err
	}
	hash = strings.TrimSpace(string(output))

	// コミットメッセージ
	cmd = exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = repoPath
	output, err = cmd.Output()
	if err != nil {
		return hash, "", time.Time{}, err
	}
	message = strings.TrimSpace(string(output))

	// コミット日時（Unix timestamp）
	cmd = exec.Command("git", "log", "-1", "--format=%ct")
	cmd.Dir = repoPath
	output, err = cmd.Output()
	if err != nil {
		return hash, message, time.Time{}, err
	}
	timestampStr := strings.TrimSpace(string(output))
	var unixTime int64
	if _, err := fmt.Sscanf(timestampStr, "%d", &unixTime); err != nil {
		return hash, message, time.Time{}, err
	}
	timestamp = time.Unix(unixTime, 0)

	return hash, message, timestamp, nil
}

// isWorkingTreeClean はワーキングツリーがクリーンかどうかを判定します
func isWorkingTreeClean(repoPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(output) == 0, nil
}

// GetDefaultBranch はデフォルトブランチを取得します
func GetDefaultBranch(repoPath string) (string, error) {
	// リモートのデフォルトブランチを取得
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		// fallback: main または master を返す
		for _, branch := range []string{"main", "master"} {
			cmd := exec.Command("git", "rev-parse", "--verify", branch)
			cmd.Dir = repoPath
			if err := cmd.Run(); err == nil {
				return branch, nil
			}
		}
		return "", err
	}

	// "refs/remotes/origin/main" -> "main"
	refPath := strings.TrimSpace(string(output))
	parts := strings.Split(refPath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1], nil
	}

	return "", &BranchNotFoundError{Branch: "default"}
}

// Pull はリポジトリをpullします
func Pull(repoPath string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitCommandError{
			Command: "git pull",
			Output:  string(output),
			Err:     err,
		}
	}
	return nil
}

// Fetch はリモートの情報を取得します
func Fetch(repoPath string, prune bool) error {
	args := []string{"fetch"}
	if prune {
		args = append(args, "--prune")
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

// Checkout はブランチを切り替えます
func Checkout(repoPath, branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitCommandError{
			Command: "git checkout " + branch,
			Output:  string(output),
			Err:     err,
		}
	}
	return nil
}

// RelativePath はベースディレクトリからの相対パスを返します
func (r *Repository) RelativePath(baseDir string) string {
	relPath, err := filepath.Rel(baseDir, r.Path)
	if err != nil {
		return r.Path
	}
	return relPath
}
