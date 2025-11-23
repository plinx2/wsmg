package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// WorkspaceShowOptions represents options for workspace show command
type WorkspaceShowOptions struct {
	Format string `flag:"format" short:"f" default:"table" usage:"Output format (table, json, yaml)"`
}

// RepoStatus represents the status of a repository in a workspace
type RepoStatus struct {
	Name             string    `json:"name" yaml:"name"`
	Path             string    `json:"path" yaml:"path"`
	RelativePath     string    `json:"relative_path" yaml:"relative_path"`
	Branch           string    `json:"branch" yaml:"branch"`
	Clean            bool      `json:"clean" yaml:"clean"`
	ModifiedFiles    int       `json:"modified_files,omitempty" yaml:"modified_files,omitempty"`
	UntrackedFiles   int       `json:"untracked_files,omitempty" yaml:"untracked_files,omitempty"`
	LastCommitHash   string    `json:"last_commit_hash,omitempty" yaml:"last_commit_hash,omitempty"`
	LastCommitAuthor string    `json:"last_commit_author,omitempty" yaml:"last_commit_author,omitempty"`
	LastCommitDate   time.Time `json:"last_commit_date,omitempty" yaml:"last_commit_date,omitempty"`
	LastCommitMsg    string    `json:"last_commit_message,omitempty" yaml:"last_commit_message,omitempty"`
}

// WorkspaceDetail represents detailed information about a workspace
type WorkspaceDetail struct {
	Name         string       `json:"name" yaml:"name"`
	Path         string       `json:"path" yaml:"path"`
	Created      time.Time    `json:"created" yaml:"created"`
	RepoCount    int          `json:"repository_count" yaml:"repository_count"`
	Repositories []RepoStatus `json:"repositories" yaml:"repositories"`
}

func newWorkspaceShowCmd() *cobra.Command {
	opts := &WorkspaceShowOptions{}

	cmd := &cobra.Command{
		Use:   "show <workspace-name>",
		Short: "Show workspace details",
		Long:  `Display detailed information about a workspace, including all repositories and their status`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runWorkspaceShow(c.Context(), args[0], opts)
		},
	}

	// Bind flags using tags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runWorkspaceShow(ctx context.Context, workspaceName string, opts *WorkspaceShowOptions) error {
	reposDir := config.GetReposDir()

	// Get workspace info
	ws, err := workspaceClient.GetWorkspace(workspaceName)
	if err != nil {
		return workspaceNotFoundError(workspaceName, err)
	}

	// Collect detailed information about repositories
	detail := WorkspaceDetail{
		Name:         ws.Name,
		Path:         ws.Path,
		Created:      ws.Created,
		RepoCount:    len(ws.Repositories),
		Repositories: make([]RepoStatus, 0, len(ws.Repositories)),
	}

	for _, r := range ws.Repositories {
		status := RepoStatus{
			Name:         r.Name(),
			Path:         r.Path(),
			RelativePath: r.RelativePath(reposDir),
		}

		// Get branch
		if branch, err := r.Branch(); err == nil {
			status.Branch = branch
		}

		// Check if clean
		if clean, err := r.IsClean(); err == nil {
			status.Clean = clean

			// If not clean, get details
			if !clean {
				if modFiles, err := r.GetModifiedFiles(); err == nil {
					status.ModifiedFiles = len(modFiles)
				}
				if untrackedFiles, err := r.GetUntrackedFiles(); err == nil {
					status.UntrackedFiles = len(untrackedFiles)
				}
			}
		}

		// Get last commit
		if commit, err := r.LastCommit(); err == nil {
			status.LastCommitHash = commit.Hash()
			status.LastCommitAuthor = commit.Author()
			status.LastCommitDate = commit.Timestamp()
			status.LastCommitMsg = commit.Message()
		}

		detail.Repositories = append(detail.Repositories, status)
	}

	// Output based on format
	switch opts.Format {
	case "json":
		return outputShowJSON(detail)
	case "yaml":
		return outputShowYAML(detail)
	default:
		return outputShowTable(detail)
	}
}

func outputShowTable(detail WorkspaceDetail) error {
	fmt.Printf("Workspace: %s\n", detail.Name)
	fmt.Printf("Path: %s\n", detail.Path)
	fmt.Printf("Created: %s\n", detail.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("Repositories: %d\n\n", detail.RepoCount)

	if len(detail.Repositories) == 0 {
		fmt.Println("No repositories in this workspace")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "NAME\tBRANCH\tSTATUS\tCHANGES\tLAST COMMIT\n")

	for _, repo := range detail.Repositories {
		status := "clean"
		changes := "-"

		if !repo.Clean {
			status = "dirty"
			changesParts := []string{}
			if repo.ModifiedFiles > 0 {
				changesParts = append(changesParts, fmt.Sprintf("%dM", repo.ModifiedFiles))
			}
			if repo.UntrackedFiles > 0 {
				changesParts = append(changesParts, fmt.Sprintf("%dU", repo.UntrackedFiles))
			}
			if len(changesParts) > 0 {
				changes = changesParts[0]
				for i := 1; i < len(changesParts); i++ {
					changes += ", " + changesParts[i]
				}
			}
		}

		lastCommit := "-"
		if repo.LastCommitHash != "" {
			shortHash := repo.LastCommitHash
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}

			timeAgo := formatTimeAgo(repo.LastCommitDate)
			lastCommit = fmt.Sprintf("%s (%s)", shortHash, timeAgo)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			repo.Name,
			repo.Branch,
			status,
			changes,
			lastCommit,
		)
	}

	return nil
}

func outputShowJSON(detail WorkspaceDetail) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(detail)
}

func outputShowYAML(detail WorkspaceDetail) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(detail)
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		return fmt.Sprintf("%dw ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		return fmt.Sprintf("%dM ago", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		return fmt.Sprintf("%dy ago", years)
	}
}

func init() {
	workspaceCmd.AddCommand(newWorkspaceShowCmd())
}
