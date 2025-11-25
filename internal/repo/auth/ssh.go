package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
)

// NewSSHAuthMethod creates a new SSH authentication method for a given endpoint
func NewSSHAuthMethod(endpoint transport.Endpoint) (ssh.AuthMethod, error) {
	// Check if URL is SSH
	if endpoint.Scheme != "ssh" {
		return nil, fmt.Errorf("not an SSH URL: %s", endpoint.String())
	}

	// Try to use SSH agent first
	authMethod, err := ssh.NewSSHAgentAuth("git")
	if err == nil {
		return authMethod, nil
	}

	// Try to use identity file
	identityFile := ssh.DefaultSSHConfig.Get(endpoint.Hostname(), "IdentityFile")
	if identityFile == "" {
		return nil, fmt.Errorf("no identity file found for host: %s", endpoint.Hostname())
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	identityFile = strings.Replace(identityFile, "~", home, 1)
	if _, err := os.Stat(identityFile); err != nil {
		return nil, fmt.Errorf("identity file not found: %s", identityFile)
	}

	publicKeyAuth, err := ssh.NewPublicKeysFromFile(endpoint.User.Username(), identityFile, "")
	if err != nil {
		return nil, err
	}
	return publicKeyAuth, nil
}
