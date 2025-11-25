package repo

import (
	"strings"
)

// Worktree はGit worktreeを表します
type Worktree struct {
	Path   string // worktreeのパス
	Branch string // ブランチ名
	Commit string // コミットハッシュ
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
