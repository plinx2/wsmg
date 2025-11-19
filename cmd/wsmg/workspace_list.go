package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/workspace"
	"github.com/spf13/cobra"
)

func newWorkspaceListCmd() *cobra.Command {
	// Define options
	opts := &struct {
		Format  string `flag:"format" default:"table" usage:"Output format" choices:"table,json,yaml"`
		Details bool   `flag:"details" short:"d" default:"false" usage:"Show detailed information"`
	}{}

	// Define command
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Long:  `Display a list of created workspaces.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runWorkspaceList(opts.Format, opts.Details)
		},
	}

	// Bind flags (auto-generated from tags)
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runWorkspaceList(format string, details bool) error {
	workspacesDir := config.GetWorkspacesDir()

	// ディレクトリの存在確認
	if _, err := os.Stat(workspacesDir); os.IsNotExist(err) {
		fmt.Printf("workspaces ディレクトリが存在しません: %s\n", workspacesDir)
		fmt.Println("ワークスペースを作成するには 'wsmg workspace create' を実行してください。")
		return nil
	}

	// ワークスペースを検索
	workspaces, err := workspace.FindWorkspaces(workspacesDir)
	if err != nil {
		return fmt.Errorf("ワークスペースの検索に失敗しました: %w", err)
	}

	if len(workspaces) == 0 {
		fmt.Println("ワークスペースが見つかりませんでした。")
		fmt.Println("ワークスペースを作成するには 'wsmg workspace create <name>' を実行してください。")
		return nil
	}

	// 出力
	switch format {
	case "json":
		return outputWorkspaceJSON(workspaces, details)
	case "yaml":
		return outputWorkspaceYAML(workspaces, details)
	default:
		return outputWorkspaceTable(workspaces, details)
	}
}

func outputWorkspaceTable(workspaces []*workspace.Workspace, details bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// ヘッダー
	if details {
		fmt.Fprintln(w, "NAME\tREPOSITORIES\tLAST MODIFIED\tPATH")
		fmt.Fprintln(w, "----\t------------\t-------------\t----")
	} else {
		fmt.Fprintln(w, "NAME\tREPOSITORIES\tLAST MODIFIED")
		fmt.Fprintln(w, "----\t------------\t-------------")
	}

	// データ行
	for _, ws := range workspaces {
		timeStr := ws.Modified.Format("2006-01-02 15:04")
		repoCount := len(ws.Repositories)

		if details {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
				ws.Name,
				repoCount,
				timeStr,
				ws.Path,
			)
		} else {
			fmt.Fprintf(w, "%s\t%d\t%s\n",
				ws.Name,
				repoCount,
				timeStr,
			)
		}

		// 詳細モードの場合、含まれるリポジトリを表示
		if details && repoCount > 0 {
			for _, repo := range ws.Repositories {
				fmt.Fprintf(w, "  └─ %s\t(%s)\t\t\n", repo.Name, repo.Branch)
			}
		}
	}

	fmt.Fprintf(w, "\nTotal: %d workspaces\n", len(workspaces))

	return nil
}

func outputWorkspaceJSON(workspaces []*workspace.Workspace, details bool) error {
	type WorkspaceJSON struct {
		Name         string                        `json:"name"`
		Path         string                        `json:"path"`
		Repositories []workspace.RepositoryInfo    `json:"repositories,omitempty"`
		RepoCount    int                           `json:"repositoryCount"`
		Modified     string                        `json:"lastModified"`
	}

	var output []WorkspaceJSON
	for _, ws := range workspaces {
		item := WorkspaceJSON{
			Name:      ws.Name,
			Path:      ws.Path,
			RepoCount: len(ws.Repositories),
			Modified:  ws.Modified.Format("2006-01-02T15:04:05Z07:00"),
		}

		if details {
			item.Repositories = ws.Repositories
		}

		output = append(output, item)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputWorkspaceYAML(workspaces []*workspace.Workspace, details bool) error {
	fmt.Println("workspaces:")
	for _, ws := range workspaces {
		fmt.Printf("  - name: %s\n", ws.Name)
		fmt.Printf("    path: %s\n", ws.Path)
		fmt.Printf("    repositoryCount: %d\n", len(ws.Repositories))
		fmt.Printf("    lastModified: %s\n", ws.Modified.Format("2006-01-02T15:04:05Z07:00"))

		if details && len(ws.Repositories) > 0 {
			fmt.Println("    repositories:")
			for _, repo := range ws.Repositories {
				fmt.Printf("      - name: %s\n", repo.Name)
				fmt.Printf("        branch: %s\n", repo.Branch)
				fmt.Printf("        path: %s\n", repo.RelativePath)
			}
		}
	}

	fmt.Printf("\ntotal: %d\n", len(workspaces))
	return nil
}
