package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/plinx2/wsmg/internal/workspace"
	"github.com/spf13/cobra"
)

// WorkspaceListOptions represents options for workspace list command
type WorkspaceListOptions struct {
	Format  string `flag:"format" default:"table" usage:"Output format" choices:"table,json,yaml"`
	Details bool   `flag:"details" short:"d" default:"false" usage:"Show detailed information"`
}

func newWorkspaceListCmd() *cobra.Command {
	// Define options
	opts := &WorkspaceListOptions{}

	// Define command
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Long:  `Display a list of created workspaces.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runWorkspaceList(c.Context(), opts)
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

func runWorkspaceList(ctx context.Context, opts *WorkspaceListOptions) error {
	workspacesDir := config.GetWorkspacesDir()

	// Check if directory exists
	if _, err := os.Stat(workspacesDir); os.IsNotExist(err) {
		fmt.Printf("workspaces directory does not exist: %s\n", workspacesDir)
		fmt.Println("Run 'wsmg workspace create' to create a workspace.")
		return nil
	}

	// List workspaces
	workspaces, err := workspaceClient.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(workspaces) == 0 {
		fmt.Println("No workspaces found.")
		fmt.Println("Run 'wsmg workspace create <name>' to create a workspace.")
		return nil
	}

	// Output
	switch opts.Format {
	case "json":
		return outputWorkspaceJSON(workspaces, opts.Details)
	case "yaml":
		return outputWorkspaceYAML(workspaces, opts.Details)
	default:
		return outputWorkspaceTable(workspaces, opts.Details)
	}
}

func outputWorkspaceTable(workspaces []*workspace.Workspace, details bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	if details {
		fmt.Fprintln(w, "NAME\tREPOSITORIES\tLAST MODIFIED\tPATH")
		fmt.Fprintln(w, "----\t------------\t-------------\t----")
	} else {
		fmt.Fprintln(w, "NAME\tREPOSITORIES\tLAST MODIFIED")
		fmt.Fprintln(w, "----\t------------\t-------------")
	}

	// Data rows
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

		// In details mode, show repositories included in the workspace
		if details && repoCount > 0 {
			for _, r := range ws.Repositories {
				branch, _ := r.Branch()
				fmt.Fprintf(w, "  └─ %s\t(%s)\t\t\n", r.Name(), branch)
			}
		}
	}

	fmt.Fprintf(w, "\nTotal: %d workspaces\n", len(workspaces))

	return nil
}

func outputWorkspaceJSON(workspaces []*workspace.Workspace, details bool) error {
	type RepoInfo struct {
		Name   string `json:"name"`
		Path   string `json:"path"`
		Branch string `json:"branch"`
	}

	type WorkspaceJSON struct {
		Name         string     `json:"name"`
		Path         string     `json:"path"`
		Repositories []RepoInfo `json:"repositories,omitempty"`
		RepoCount    int        `json:"repositoryCount"`
		Modified     string     `json:"lastModified"`
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
			for _, r := range ws.Repositories {
				branch, _ := r.Branch()
				item.Repositories = append(item.Repositories, RepoInfo{
					Name:   r.Name(),
					Path:   r.Path(),
					Branch: branch,
				})
			}
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
			for _, r := range ws.Repositories {
				branch, _ := r.Branch()
				fmt.Printf("      - name: %s\n", r.Name())
				fmt.Printf("        branch: %s\n", branch)
				fmt.Printf("        path: %s\n", r.Path())
			}
		}
	}

	fmt.Printf("\ntotal: %d\n", len(workspaces))
	return nil
}
