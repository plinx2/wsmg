package git

import "fmt"

// NotRepositoryError はディレクトリがGitリポジトリでない場合のエラーです
type NotRepositoryError struct {
	Path string
}

func (e *NotRepositoryError) Error() string {
	return fmt.Sprintf("not a git repository: %s", e.Path)
}

// BranchNotFoundError はブランチが見つからない場合のエラーです
type BranchNotFoundError struct {
	Branch string
}

func (e *BranchNotFoundError) Error() string {
	return fmt.Sprintf("branch not found: %s", e.Branch)
}

// GitCommandError はGitコマンドの実行エラーです
type GitCommandError struct {
	Command string
	Output  string
	Err     error
}

func (e *GitCommandError) Error() string {
	return fmt.Sprintf("git command failed: %s\n%s\n%v", e.Command, e.Output, e.Err)
}

func (e *GitCommandError) Unwrap() error {
	return e.Err
}
