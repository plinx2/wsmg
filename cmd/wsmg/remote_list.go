package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/remote"
	"github.com/plinx2/wsmg/internal/remote/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// RemoteListOptions represents options for remote list command
type RemoteListOptions struct {
	Provider     string   `flag:"provider" short:"p" default:"github" usage:"Provider name (github, gitlab)"`
	Organization string   `flag:"org" short:"o" default:"" usage:"Organization/Group name"`
	Topics       []string `flag:"topic" short:"t" usage:"Filter by topics"`
	Archived     bool     `flag:"archived" default:"false" usage:"Include archived repositories"`
	Type         string   `flag:"type" default:"all" usage:"Repository type (all, owner, member, public, private)"`
	Language     string   `flag:"language" short:"l" default:"" usage:"Filter by programming language"`
	Sort         string   `flag:"sort" default:"updated" usage:"Sort by (updated, created, pushed, full_name)"`
	Direction    string   `flag:"direction" default:"desc" usage:"Sort direction (asc, desc)"`
	Limit        int      `flag:"limit" default:"0" usage:"Maximum number of repositories (0 for unlimited)"`
	Format       string   `flag:"format" short:"f" default:"table" usage:"Output format (table, json, yaml)"`
}

func newRemoteListCmd() *cobra.Command {
	opts := &RemoteListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List remote repositories",
		Long:  `List repositories from remote providers (GitHub, GitLab, etc.)`,
		RunE: func(c *cobra.Command, args []string) error {
			return runRemoteList(c.Context(), opts)
		},
	}

	// Bind flags using tags
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	// Register completions
	cmd.RegisterFlagCompletionFunc("format", formatCompletion)
	cmd.RegisterFlagCompletionFunc("provider", providerCompletion)

	return cmd
}

func runRemoteList(ctx context.Context, opts *RemoteListOptions) error {
	// Get provider client
	provider, err := getProvider(opts.Provider)
	if err != nil {
		return err
	}

	// List repositories
	fmt.Fprintf(os.Stderr, "Fetching repositories from %s...\n", opts.Provider)

	repos, err := provider.ListRepositories(ctx, remote.ListOptions{
		Organization: opts.Organization,
		Topics:       opts.Topics,
		Archived:     opts.Archived,
		Type:         opts.Type,
		Language:     opts.Language,
		Sort:         opts.Sort,
		Direction:    opts.Direction,
		Limit:        opts.Limit,
	})
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found")
		return nil
	}

	// Output results
	switch opts.Format {
	case "json":
		return outputRemoteJSON(repos)
	case "yaml":
		return outputRemoteYAML(repos)
	default:
		return outputRemoteTable(repos)
	}
}

func getProvider(providerName string) (remote.Provider, error) {
	switch strings.ToLower(providerName) {
	case "github":
		// Get GitHub token from config or environment
		token := viper.GetString("github.token")
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}
		if token == "" {
			return nil, fmt.Errorf("GitHub token not found. Set GITHUB_TOKEN environment variable or configure in wsmg.json")
		}

		return github.NewClient(token)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

func outputRemoteTable(repos []*remote.Repository) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintf(w, "FULL NAME\tLANGUAGE\tSTARS\tPRIVATE\tARCHIVED\tUPDATED\n")

	// Rows
	for _, repo := range repos {
		language := repo.Language
		if language == "" {
			language = "-"
		}

		private := "no"
		if repo.Private {
			private = "yes"
		}

		archived := "no"
		if repo.Archived {
			archived = "yes"
		}

		updated := formatTime(repo.UpdatedAt)

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			repo.FullName,
			language,
			repo.Stars,
			private,
			archived,
			updated,
		)
	}

	fmt.Fprintf(os.Stderr, "\nTotal: %d repositories\n", len(repos))
	return nil
}

func outputRemoteJSON(repos []*remote.Repository) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(repos)
}

func outputRemoteYAML(repos []*remote.Repository) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(repos)
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 0 {
			return "just now"
		}
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
	remoteCmd.AddCommand(newRemoteListCmd())
}
