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

func newWorkspaceDeleteCmd() *cobra.Command {
	// Define options
	opts := &struct {
		Force         bool `flag:"force" short:"f" default:"false" usage:"Force delete even with uncommitted changes"`
		KeepBranches  bool `flag:"keep-branches" default:"false" usage:"Delete only worktrees, keep branches"`
		DeleteRemote  bool `flag:"delete-remote" default:"false" usage:"Delete remote branches as well"`
	}{}

	// Define command
	cmd := &cobra.Command{
		Use:   "delete <workspace-name>",
		Short: "Delete a workspace",
		Long:  `Delete a workspace, including Git worktrees and branches.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			workspaceName := args[0]
			return runWorkspaceDelete(workspaceName, opts.Force, opts.KeepBranches, opts.DeleteRemote)
		},
	}

	// Bind flags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runWorkspaceDelete(workspaceName string, force, keepBranches, deleteRemote bool) error {
	reposDir := config.GetReposDir()
	workspacesDir := config.GetWorkspacesDir()

	// Check if workspace exists
	workspacePath := filepath.Join(workspacesDir, workspaceName)
	if !workspace.WorkspaceExists(workspacesDir, workspaceName) {
		return fmt.Errorf("workspace '%s' not found", workspaceName)
	}

	// Get workspace info
	ws, err := workspace.GetWorkspaceInfo(workspacePath, workspaceName)
	if err != nil {
		return fmt.Errorf("failed to get workspace info: %w", err)
	}

	fmt.Printf("Workspace: %s\n", workspaceName)
	fmt.Printf("Path: %s\n", workspacePath)
	fmt.Printf("Repositories: %d\n", len(ws.Repositories))

	if len(ws.Repositories) > 0 {
		fmt.Println("\nRepositories:")
		for _, repo := range ws.Repositories {
			fmt.Printf("  - %s (branch: %s)\n", repo.Name, repo.Branch)
		}
	}

	// 確認
	if !force {
		fmt.Print("\nワークスペースを削除しますか? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("キャンセルしました。")
			return nil
		}
	}

	fmt.Printf("\nワークスペースを削除しています...\n\n")

	// 各リポジトリのworktreeとブランチを削除
	successCount := 0
	errorCount := 0

	for _, repoInfo := range ws.Repositories {
		fmt.Printf("Processing %s...", repoInfo.Name)

		// リポジトリの本体パスを構築
		repoPath := filepath.Join(reposDir, repoInfo.RelativePath)

		// worktreeパス
		worktreePath := filepath.Join(workspacePath, repoInfo.RelativePath)

		// worktreeを削除
		if err := git.RemoveWorktree(repoPath, worktreePath); err != nil {
			fmt.Printf(" ✗ worktree removal failed: %v\n", err)
			errorCount++
			continue
		}

		// ブランチを削除（オプション）
		if !keepBranches {
			if err := git.DeleteBranch(repoPath, repoInfo.Branch, force); err != nil {
				// ブランチ削除の失敗は警告として扱う
				fmt.Printf(" ⚠ branch deletion failed: %v\n", err)
			} else {
				// リモートブランチも削除（オプション）
				if deleteRemote {
					// TODO: リモートブランチの削除を実装
					// git push origin --delete <branch>
				}
				fmt.Println(" ✓")
			}
		} else {
			fmt.Println(" ✓ (branch kept)")
		}

		successCount++
	}

	// ワークスペースディレクトリを削除
	fmt.Print("\nRemoving workspace directory...")
	if err := workspace.DeleteWorkspace(workspacePath); err != nil {
		fmt.Printf(" ✗ failed: %v\n", err)
		return err
	}
	fmt.Println(" ✓")

	// サマリーを表示
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Processed: %d repositories\n", successCount)
	if errorCount > 0 {
		fmt.Printf("  Errors:    %d repositories\n", errorCount)
	}
	fmt.Printf("\nWorkspace deleted: %s\n", workspaceName)

	return nil
}
