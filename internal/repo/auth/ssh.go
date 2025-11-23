package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
)

// DetectSSHAuthMethod detects and returns the appropriate SSH authentication method for a given URL
func DetectSSHAuthMethod(url string) (ssh.AuthMethod, error) {
	// Check if URL is SSH
	if !strings.HasPrefix(url, "git@") && !strings.HasPrefix(url, "ssh://") {
		return nil, fmt.Errorf("not an SSH URL: %s", url)
	}

	// Try to use SSH agent first
	authMethod, err := ssh.NewSSHAgentAuth("git")
	if err == nil {
		return authMethod, nil
	}

	// Fallback to SSH key file
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Try common SSH key locations
	keyPaths := []string{
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	}

	for _, keyPath := range keyPaths {
		if _, err := os.Stat(keyPath); err == nil {
			authMethod, err := ssh.NewPublicKeysFromFile("git", keyPath, "")
			if err == nil {
				return authMethod, nil
			}
		}
	}

	return nil, fmt.Errorf("no SSH authentication method available")
}
