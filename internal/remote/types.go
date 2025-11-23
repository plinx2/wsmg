package remote

import "time"

// Repository represents a remote repository
type Repository struct {
	Provider    string    `json:"provider"`    // "github", "gitlab", etc.
	Owner       string    `json:"owner"`       // Repository owner/organization
	Name        string    `json:"name"`        // Repository name
	FullName    string    `json:"full_name"`   // "owner/repo"
	SSHURL      string    `json:"ssh_url"`     // SSH clone URL
	HTTPSURL    string    `json:"https_url"`   // HTTPS clone URL
	Description string    `json:"description"` // Repository description
	Private     bool      `json:"private"`     // Is private repository
	Archived    bool      `json:"archived"`    // Is archived
	Language    string    `json:"language"`    // Primary language
	Stars       int       `json:"stars"`       // Star count
	Forks       int       `json:"forks"`       // Fork count
	Topics      []string  `json:"topics"`      // Repository topics/tags
	UpdatedAt   time.Time `json:"updated_at"`  // Last update time
	CreatedAt   time.Time `json:"created_at"`  // Creation time
}

// ListOptions represents options for listing repositories
type ListOptions struct {
	Organization string   // Filter by organization/group
	Topics       []string // Filter by topics
	Archived     bool     // Include archived repos (default: false)
	Type         string   // "all", "owner", "member", "public", "private"
	Language     string   // Filter by programming language
	Sort         string   // "updated", "created", "pushed", "full_name"
	Direction    string   // "asc", "desc"
	Limit        int      // Maximum number of repositories to return
}

// CloneOptions represents options for cloning a repository
type CloneOptions struct {
	UseSSH bool   // Use SSH URL instead of HTTPS
	Branch string // Branch to checkout (empty for default)
	Depth  int    // Clone depth (0 for full clone)
}
