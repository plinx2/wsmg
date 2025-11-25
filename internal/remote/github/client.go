package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/go-github/v66/github"
	"github.com/plinx2/wsmg/internal/remote"
	"golang.org/x/oauth2"
)

// Client represents a GitHub API client
type Client struct {
	client *github.Client
	token  string
}

// NewClient creates a new GitHub client
func NewClient(token string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: github.NewClient(tc),
		token:  token,
	}, nil
}

// Name returns the provider name
func (c *Client) Name() string {
	return "github"
}

// ListRepositories lists all accessible repositories
func (c *Client) ListRepositories(ctx context.Context, opts remote.ListOptions) ([]*remote.Repository, error) {
	var allRepos []*remote.Repository

	// Determine list options
	listOpts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	// Set type filter
	switch opts.Type {
	case "owner":
		listOpts.Affiliation = "owner"
	case "member":
		listOpts.Affiliation = "collaborator,organization_member"
	case "public":
		listOpts.Visibility = "public"
	case "private":
		listOpts.Visibility = "private"
	default:
		listOpts.Affiliation = "owner,collaborator,organization_member"
	}

	// Set sort options
	if opts.Sort != "" {
		listOpts.Sort = opts.Sort
	} else {
		listOpts.Sort = "updated"
	}

	if opts.Direction != "" {
		listOpts.Direction = opts.Direction
	} else {
		listOpts.Direction = "desc"
	}

	// If organization is specified, list org repositories
	if opts.Organization != "" {
		for {
			repos, resp, err := c.client.Repositories.ListByOrg(ctx, opts.Organization, &github.RepositoryListByOrgOptions{
				Type:        "all",
				Sort:        listOpts.Sort,
				Direction:   listOpts.Direction,
				ListOptions: listOpts.ListOptions,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to list organization repositories: %w", err)
			}

			for _, repo := range repos {
				r := c.convertRepository(repo)
				if c.matchesFilters(r, opts) {
					allRepos = append(allRepos, r)
				}
			}

			if resp.NextPage == 0 {
				break
			}
			listOpts.Page = resp.NextPage

			// Check limit
			if opts.Limit > 0 && len(allRepos) >= opts.Limit {
				allRepos = allRepos[:opts.Limit]
				break
			}
		}
	} else {
		// List user repositories
		for {
			repos, resp, err := c.client.Repositories.List(ctx, "", listOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to list repositories: %w", err)
			}

			for _, repo := range repos {
				r := c.convertRepository(repo)
				if c.matchesFilters(r, opts) {
					allRepos = append(allRepos, r)
				}
			}

			if resp.NextPage == 0 {
				break
			}
			listOpts.Page = resp.NextPage

			// Check limit
			if opts.Limit > 0 && len(allRepos) >= opts.Limit {
				allRepos = allRepos[:opts.Limit]
				break
			}
		}
	}

	return allRepos, nil
}

// GetRepository gets repository details
func (c *Client) GetRepository(ctx context.Context, owner, name string) (*remote.Repository, error) {
	repo, _, err := c.client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return c.convertRepository(repo), nil
}

// Clone clones a repository to the specified destination
func (c *Client) Clone(ctx context.Context, repo *remote.Repository, destPath string, opts remote.CloneOptions) error {
	// Determine clone URL
	cloneURL := repo.HTTPSURL
	if opts.UseSSH {
		cloneURL = repo.SSHURL
	}

	// Build git clone command
	args := []string{"clone"}

	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}

	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}

	args = append(args, cloneURL, destPath)

	// Execute git clone
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// convertRepository converts GitHub repository to remote.Repository
func (c *Client) convertRepository(repo *github.Repository) *remote.Repository {
	r := &remote.Repository{
		Provider:    "github",
		Owner:       repo.GetOwner().GetLogin(),
		Name:        repo.GetName(),
		FullName:    repo.GetFullName(),
		SSHURL:      repo.GetSSHURL(),
		HTTPSURL:    repo.GetCloneURL(),
		Description: repo.GetDescription(),
		Private:     repo.GetPrivate(),
		Archived:    repo.GetArchived(),
		Language:    repo.GetLanguage(),
		Stars:       repo.GetStargazersCount(),
		Forks:       repo.GetForksCount(),
		Topics:      repo.Topics,
		UpdatedAt:   repo.GetUpdatedAt().Time,
		CreatedAt:   repo.GetCreatedAt().Time,
	}

	return r
}

// matchesFilters checks if repository matches the filter criteria
func (c *Client) matchesFilters(repo *remote.Repository, opts remote.ListOptions) bool {
	// Filter by archived status
	if !opts.Archived && repo.Archived {
		return false
	}

	// Filter by language
	if opts.Language != "" && repo.Language != opts.Language {
		return false
	}

	// Filter by topics
	if len(opts.Topics) > 0 {
		hasAllTopics := true
		for _, topic := range opts.Topics {
			found := false
			for _, repoTopic := range repo.Topics {
				if repoTopic == topic {
					found = true
					break
				}
			}
			if !found {
				hasAllTopics = false
				break
			}
		}
		if !hasAllTopics {
			return false
		}
	}

	return true
}
