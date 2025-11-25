package main

import (
	"context"
	"fmt"
	"os"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/repo"
	"github.com/spf13/cobra"
)

// ReposSyncOptions はリポジトリ同期コマンドのオプションです
type ReposSyncOptions struct {
	Filter string `flag:"filter" short:"f" default:"" usage:"同期対象のリポジトリをフィルタリング"`
	Prune  bool   `flag:"prune" short:"p" default:"false" usage:"リモートで削除されたブランチをローカルからも削除"`
	DryRun bool   `flag:"dry-run" short:"n" default:"false" usage:"実際には同期せず、対象リポジトリのみ表示"`
}

func newReposSyncCmd() *cobra.Command {
	// オプション定義
	opts := &ReposSyncOptions{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync repositories",
		Long:  `repos内のリポジトリのデフォルトブランチを最新に更新します。`,
		RunE: func(c *cobra.Command, args []string) error {
			return runReposSync(c.Context(), opts)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runReposSync(ctx context.Context, opts *ReposSyncOptions) error {
	reposDir := config.GetReposDir()

	// Find repositories
	fmt.Fprintf(os.Stderr, "Scanning repositories in %s...\n", reposDir)

	repos, err := repoClient.List(repo.ListInput{
		Filter: opts.Filter,
		Sort:   repo.ListSortFieldPath,
	})
	if err != nil {
		return fmt.Errorf("failed to find repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	fmt.Printf("Found %d repositories\n\n", len(repos))

	// Dry-run mode
	if opts.DryRun {
		fmt.Println("Dry-run mode: The following repositories would be synced:")
		for _, r := range repos {
			branch, _ := r.Branch()
			fmt.Printf("  - %s (branch: %s)\n", r.RelativePath(reposDir), branch)
		}
		return nil
	}

	// Sync repositories
	if err := workspaceClient.SyncRepositories(ctx, opts.Filter, opts.Prune); err != nil {
		return err
	}

	fmt.Println("\nSync completed successfully!")
	return nil
}
