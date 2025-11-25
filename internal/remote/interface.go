package remote

import "context"

// Provider represents a remote repository provider (GitHub, GitLab, etc.)
type Provider interface {
	// Name returns the provider name
	Name() string

	// ListRepositories lists all accessible repositories
	ListRepositories(ctx context.Context, opts ListOptions) ([]*Repository, error)

	// GetRepository gets repository details
	GetRepository(ctx context.Context, owner, repo string) (*Repository, error)

	// Clone clones a repository to the specified destination
	Clone(ctx context.Context, repo *Repository, destPath string, opts CloneOptions) error
}
