package repo

import (
	"time"

	"github.com/go-git/go-git/v6/plumbing/object"
)

// Commit represents a git commit
type Commit struct {
	commit *object.Commit
}

func (c *Commit) Hash() string {
	return c.commit.Hash.String()[:7]
}

func (c *Commit) Message() string {
	lines := c.commit.Message
	if idx := len(lines); idx > 0 {
		if lines[idx-1] == '\n' {
			lines = lines[:idx-1]
		}
	}
	return lines
}

func (c *Commit) Timestamp() time.Time {
	return c.commit.Author.When
}

func (c *Commit) Author() string {
	return c.commit.Author.Name
}
