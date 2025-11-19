package main

import (
	"fmt"
	"os"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/git"
	"github.com/spf13/cobra"
)

func newReposSyncCmd() *cobra.Command {
	// オプション定義
	opts := &struct {
		Filter string `flag:"filter" short:"f" default:"" usage:"同期対象のリポジトリをフィルタリング"`
		Prune  bool   `flag:"prune" short:"p" default:"false" usage:"リモートで削除されたブランチをローカルからも削除"`
		DryRun bool   `flag:"dry-run" short:"n" default:"false" usage:"実際には同期せず、対象リポジトリのみ表示"`
	}{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync repositories",
		Long:  `repos内のリポジトリのデフォルトブランチを最新に更新します。`,
		RunE: func(c *cobra.Command, args []string) error {
			return runReposSync(opts.Filter, opts.Prune, opts.DryRun)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runReposSync(filter string, prune, dryRun bool) error {
	reposDir := config.GetReposDir()

	// ディレクトリの存在確認
	if _, err := os.Stat(reposDir); os.IsNotExist(err) {
		return fmt.Errorf("repos ディレクトリが存在しません: %s", reposDir)
	}

	// リポジトリを検索
	fmt.Fprintf(os.Stderr, "Scanning repositories in %s...\n", reposDir)
	repos, err := git.FindRepositories(reposDir)
	if err != nil {
		return fmt.Errorf("リポジトリの検索に失敗しました: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("リポジトリが見つかりませんでした。")
		return nil
	}

	// フィルタリング
	if filter != "" {
		repos = filterRepositories(repos, filter)
	}

	if len(repos) == 0 {
		fmt.Println("条件に一致するリポジトリが見つかりませんでした。")
		return nil
	}

	fmt.Printf("Found %d repositories\n\n", len(repos))

	// dry-run モード
	if dryRun {
		fmt.Println("Dry-run mode: The following repositories would be synced:")
		for _, repo := range repos {
			fmt.Printf("  - %s (branch: %s)\n", repo.RelativePath(reposDir), repo.Branch)
		}
		return nil
	}

	// 各リポジトリを同期
	successCount := 0
	errorCount := 0

	for _, repo := range repos {
		relPath := repo.RelativePath(reposDir)
		fmt.Printf("Syncing %s...", relPath)

		// fetch を実行
		if err := git.Fetch(repo.Path, prune); err != nil {
			fmt.Printf(" ✗ fetch failed: %v\n", err)
			errorCount++
			continue
		}

		// デフォルトブランチを取得
		defaultBranch, err := git.GetDefaultBranch(repo.Path)
		if err != nil {
			fmt.Printf(" ✗ failed to get default branch: %v\n", err)
			errorCount++
			continue
		}

		// 現在のブランチがデフォルトブランチでない場合はチェックアウト
		if repo.Branch != defaultBranch {
			fmt.Printf(" (checkout %s)", defaultBranch)
			if err := git.Checkout(repo.Path, defaultBranch); err != nil {
				fmt.Printf(" ✗ checkout failed: %v\n", err)
				errorCount++
				continue
			}
		}

		// pull を実行
		if err := git.Pull(repo.Path); err != nil {
			fmt.Printf(" ✗ pull failed: %v\n", err)
			errorCount++
			continue
		}

		fmt.Println(" ✓")
		successCount++
	}

	// サマリーを表示
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Success: %d\n", successCount)
	if errorCount > 0 {
		fmt.Printf("  Error:   %d\n", errorCount)
		return fmt.Errorf("%d repositories failed to sync", errorCount)
	}

	return nil
}
