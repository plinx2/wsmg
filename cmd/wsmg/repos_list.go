package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/repo"
	"github.com/spf13/cobra"
)

// ReposListOptions はリポジトリ一覧コマンドのオプションです
type ReposListOptions struct {
	Filter string `flag:"filter" short:"f" default:"" usage:"Filter by repository name or path"`
	Format string `flag:"format" default:"table" usage:"Output format" choices:"table,json,yaml"`
	Sort   string `flag:"sort" default:"path" usage:"Sort field" choices:"path,name,lastcommit"`
}

func newReposListCmd() *cobra.Command {
	// Define options
	opts := &ReposListOptions{}

	// Define command
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		Long:  `Display a list of repositories in repos directory with path, branch, last commit, and other information.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runReposList(c.Context(), opts)
		},
	}

	// Bind flags (auto-generated from tags)
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	// Register completions
	cmd.RegisterFlagCompletionFunc("format", formatCompletion)

	return cmd
}

func runReposList(ctx context.Context, opts *ReposListOptions) error {
	reposDir := config.GetReposDir()

	// Find repositories
	fmt.Fprintf(os.Stderr, "Scanning repositories in %s...\n", reposDir)

	// Convert sort field
	var sortInput repo.ListSortField
	switch opts.Sort {
	case "name":
		sortInput = repo.ListSortFieldName
	case "lastcommit":
		sortInput = repo.ListSortFieldLastCommit
	default:
		sortInput = repo.ListSortFieldPath
	}

	repos, err := repoClient.List(repo.ListInput{
		Filter: opts.Filter,
		Sort:   sortInput,
	})
	if err != nil {
		return fmt.Errorf("failed to find repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	// Output
	switch opts.Format {
	case "json":
		return outputJSON(repos, reposDir)
	case "yaml":
		return outputYAML(repos, reposDir)
	default:
		return outputTable(repos, reposDir)
	}
}

func outputTable(repos []*repo.Local, baseDir string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "PATH\tBRANCH\tLAST COMMIT\tLAST MODIFIED\tSTATUS")
	fmt.Fprintln(w, "----\t------\t-----------\t-------------\t------")

	// Data rows
	for _, r := range repos {
		relPath := r.RelativePath(baseDir)

		branch, err := r.Branch()
		if err != nil {
			branch = "N/A"
		}

		lastCommit, err := r.LastCommit()
		var commitInfo string
		var timeStr string
		if err != nil {
			commitInfo = "N/A"
			timeStr = "N/A"
		} else {
			commitInfo = fmt.Sprintf("%s %s", lastCommit.Hash(), truncate(lastCommit.Message(), 30))
			timeStr = lastCommit.Timestamp().Format("2006-01-02 15:04")
		}

		isClean, err := r.IsClean()
		status := "clean"
		if err == nil && !isClean {
			status = "dirty"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			relPath,
			branch,
			commitInfo,
			timeStr,
			status,
		)
	}

	fmt.Fprintf(w, "\nTotal: %d repositories\n", len(repos))

	return nil
}

func outputJSON(repos []*repo.Local, baseDir string) error {
	type RepoJSON struct {
		Path         string `json:"path"`
		RelativePath string `json:"relativePath"`
		Name         string `json:"name"`
		Branch       string `json:"branch"`
		LastCommit   string `json:"lastCommit"`
		LastMessage  string `json:"lastMessage"`
		LastModified string `json:"lastModified"`
		IsClean      bool   `json:"isClean"`
	}

	var output []RepoJSON
	for _, r := range repos {
		branch, _ := r.Branch()
		lastCommit, _ := r.LastCommit()
		isClean, _ := r.IsClean()

		item := RepoJSON{
			Path:         r.Path(),
			RelativePath: r.RelativePath(baseDir),
			Name:         r.Name(),
			Branch:       branch,
			IsClean:      isClean,
		}

		if lastCommit != nil {
			item.LastCommit = lastCommit.Hash()
			item.LastMessage = lastCommit.Message()
			item.LastModified = lastCommit.Timestamp().Format("2006-01-02T15:04:05Z07:00")
		}

		output = append(output, item)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputYAML(repos []*repo.Local, baseDir string) error {
	fmt.Println("repositories:")
	for _, r := range repos {
		fmt.Printf("  - path: %s\n", r.RelativePath(baseDir))
		fmt.Printf("    name: %s\n", r.Name())

		branch, _ := r.Branch()
		fmt.Printf("    branch: %s\n", branch)

		lastCommit, _ := r.LastCommit()
		if lastCommit != nil {
			fmt.Printf("    lastCommit: %s\n", lastCommit.Hash())
			fmt.Printf("    lastMessage: %s\n", lastCommit.Message())
			fmt.Printf("    lastModified: %s\n", lastCommit.Timestamp().Format("2006-01-02T15:04:05Z07:00"))
		}

		isClean, _ := r.IsClean()
		fmt.Printf("    isClean: %t\n", isClean)
	}
	fmt.Printf("\ntotal: %d\n", len(repos))
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
