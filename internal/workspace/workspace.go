package workspace

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Workspace はワークスペースを表します
type Workspace struct {
	Name         string           // ワークスペース名（チケット名）
	Path         string           // ワークスペースのパス
	Repositories []RepositoryInfo // 含まれるリポジトリ
	Created      time.Time        // 作成日時
	Modified     time.Time        // 最終更新日時
}

// RepositoryInfo はワークスペース内のリポジトリ情報を表します
type RepositoryInfo struct {
	Name         string // リポジトリ名
	RelativePath string // ワークスペースからの相対パス
	Branch       string // ブランチ名
}

// FindWorkspaces は指定されたディレクトリ配下のワークスペースを検索します
func FindWorkspaces(workspacesDir string) ([]*Workspace, error) {
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Workspace{}, nil
		}
		return nil, err
	}

	var workspaces []*Workspace
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspacePath := filepath.Join(workspacesDir, entry.Name())
		ws, err := GetWorkspaceInfo(workspacePath, entry.Name())
		if err != nil {
			// エラーは無視してスキップ
			continue
		}

		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

// GetWorkspaceInfo はワークスペースの情報を取得します
func GetWorkspaceInfo(workspacePath, name string) (*Workspace, error) {
	info, err := os.Stat(workspacePath)
	if err != nil {
		return nil, err
	}

	ws := &Workspace{
		Name:     name,
		Path:     workspacePath,
		Created:  info.ModTime(), // 簡易的に修正日時を使用
		Modified: info.ModTime(),
	}

	// ワークスペース内のリポジトリを検索
	repos, err := findRepositoriesInWorkspace(workspacePath)
	if err == nil {
		ws.Repositories = repos
	}

	return ws, nil
}

// findRepositoriesInWorkspace はワークスペース内のリポジトリを検索します
func findRepositoriesInWorkspace(workspacePath string) ([]RepositoryInfo, error) {
	var repos []RepositoryInfo

	err := filepath.Walk(workspacePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーは無視
		}

		// .git を見つけたらリポジトリとして認識（ディレクトリまたはファイル）
		if info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			relPath, err := filepath.Rel(workspacePath, repoPath)
			if err != nil {
				return nil
			}

			// ブランチ名を取得
			branch := getCurrentBranchFromGit(path, info.IsDir())

			repos = append(repos, RepositoryInfo{
				Name:         filepath.Base(repoPath),
				RelativePath: relPath,
				Branch:       branch,
			})

			if info.IsDir() {
				return filepath.SkipDir
			}
		}

		return nil
	})

	return repos, err
}

// getCurrentBranchFromGit は.gitからブランチ名を取得します
func getCurrentBranchFromGit(gitPath string, isDir bool) string {
	var headFile string

	if isDir {
		// 通常のリポジトリ
		headFile = filepath.Join(gitPath, "HEAD")
	} else {
		// worktreeの場合、.gitファイルの内容を読んで実際のHEADを見つける
		data, err := os.ReadFile(gitPath)
		if err != nil {
			return ""
		}

		// "gitdir: /path/to/.git/worktrees/xxx" の形式
		content := strings.TrimSpace(string(data))
		if strings.HasPrefix(content, "gitdir: ") {
			gitdir := strings.TrimPrefix(content, "gitdir: ")
			headFile = filepath.Join(gitdir, "HEAD")
		} else {
			return ""
		}
	}

	// HEADファイルを読む
	data, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}

	// "ref: refs/heads/main" -> "main"
	head := strings.TrimSpace(string(data))
	if strings.HasPrefix(head, "ref: refs/heads/") {
		return strings.TrimPrefix(head, "ref: refs/heads/")
	}

	return ""
}

// VSCodeWorkspace はVSCodeのワークスペース設定を表します
type VSCodeWorkspace struct {
	Folders  []VSCodeFolder         `json:"folders"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// VSCodeFolder はVSCodeのフォルダ設定を表します
type VSCodeFolder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// GenerateVSCodeWorkspace はVSCodeワークスペースファイルを生成します
func GenerateVSCodeWorkspace(workspacePath, workspaceName string, repos []RepositoryInfo) error {
	workspace := VSCodeWorkspace{
		Folders: make([]VSCodeFolder, 0, len(repos)),
		Settings: map[string]interface{}{
			"files.exclude": map[string]bool{
				"**/.git": true,
			},
		},
	}

	// リポジトリをフォルダとして追加
	for _, repo := range repos {
		workspace.Folders = append(workspace.Folders, VSCodeFolder{
			Name: repo.Name,
			Path: repo.RelativePath,
		})
	}

	// JSONファイルとして保存
	workspaceFile := filepath.Join(workspacePath, workspaceName+".code-workspace")
	data, err := json.MarshalIndent(workspace, "", "  ")
	if err != nil {
		return fmt.Errorf("JSONの生成に失敗しました: %w", err)
	}

	if err := os.WriteFile(workspaceFile, data, 0644); err != nil {
		return fmt.Errorf("ワークスペースファイルの書き込みに失敗しました: %w", err)
	}

	return nil
}

// WorkspaceExists はワークスペースが存在するかチェックします
func WorkspaceExists(workspacesDir, name string) bool {
	workspacePath := filepath.Join(workspacesDir, name)
	info, err := os.Stat(workspacePath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// CreateWorkspaceDir はワークスペースディレクトリを作成します
func CreateWorkspaceDir(workspacesDir, name string) (string, error) {
	workspacePath := filepath.Join(workspacesDir, name)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return "", fmt.Errorf("ワークスペースディレクトリの作成に失敗しました: %w", err)
	}
	return workspacePath, nil
}

// DeleteWorkspace はワークスペースを削除します
func DeleteWorkspace(workspacePath string) error {
	if err := os.RemoveAll(workspacePath); err != nil {
		return fmt.Errorf("ワークスペースの削除に失敗しました: %w", err)
	}
	return nil
}

// CopyEnvironmentFiles は指定されたパターンに一致するファイル・ディレクトリを
// 元のリポジトリから worktree にコピーします
// パターンはプロジェクトルートからの相対パスに対してマッチングされます
func CopyEnvironmentFiles(sourceRepo, targetWorktree string, patterns []string) ([]string, error) {
	var copiedFiles []string
	copiedPaths := make(map[string]bool) // 重複を避けるため

	// ルートディレクトリから再帰的にスキャン（最大深さ3まで）
	err := filepath.Walk(sourceRepo, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーは無視して続行
		}

		// ルートディレクトリ自体はスキップ
		if path == sourceRepo {
			return nil
		}

		// 相対パスを取得
		relPath, err := filepath.Rel(sourceRepo, path)
		if err != nil {
			return nil
		}

		// 深さをチェック（最大3階層まで）
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > 3 {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// .git ディレクトリはスキップ
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// パターンにマッチするかチェック
		if !matchesAnyPattern(relPath, patterns) {
			return nil
		}

		// 既にコピー済みのパスはスキップ
		if copiedPaths[relPath] {
			return nil
		}

		// コピー先のパス
		targetPath := filepath.Join(targetWorktree, relPath)

		if info.IsDir() {
			// ディレクトリの場合は作成
			if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
				return nil // エラーは無視
			}
			copiedFiles = append(copiedFiles, relPath+"/")
			copiedPaths[relPath] = true
		} else {
			// ファイルの場合はコピー
			// 親ディレクトリを作成
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return nil // エラーは無視
			}

			if err := copyFile(path, targetPath); err != nil {
				return nil // エラーは無視
			}
			copiedFiles = append(copiedFiles, relPath)
			copiedPaths[relPath] = true
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("ファイルのスキャンに失敗しました: %w", err)
	}

	return copiedFiles, nil
}

// matchesAnyPattern は相対パスが指定されたパターンのいずれかに一致するかチェックします
// ワイルドカード (*) をサポートします
func matchesAnyPattern(relPath string, patterns []string) bool {
	// パス区切り文字を正規化（Windows対応）
	relPath = filepath.ToSlash(relPath)

	for _, pattern := range patterns {
		// パターンも正規化
		pattern = filepath.ToSlash(pattern)

		// ディレクトリパターン (例: .vscode/*)
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			// .vscode/* は .vscode/settings.json などにマッチ
			if relPath == prefix || strings.HasPrefix(relPath, prefix+"/") {
				return true
			}
			continue
		}

		// ワイルドカードパターンマッチング
		matched, err := filepath.Match(pattern, relPath)
		if err != nil {
			// パターンエラーの場合は完全一致で試す
			if relPath == pattern {
				return true
			}
			continue
		}

		if matched {
			return true
		}

		// basename だけでもマッチングを試す（後方互換性のため）
		basename := filepath.Base(relPath)
		matched, err = filepath.Match(pattern, basename)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// copyFile はファイルをコピーします
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// ファイル情報を取得してパーミッションを保持
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// ファイルをコピー
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// パーミッションを設定
	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return err
	}

	return nil
}
