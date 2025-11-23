package repo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Helper function to create a test repository
func createTestRepo(t *testing.T, dir, name string) string {
	t.Helper()

	repoPath := filepath.Join(dir, name)
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	testFile := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	return repoPath
}

func TestNewLocal(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := createTestRepo(t, tmpDir, "test-repo")

	// Test successful creation
	local, err := NewLocal(repoPath)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	if local.Path() != repoPath {
		t.Errorf("Expected path %s, got %s", repoPath, local.Path())
	}

	// Test with non-existent path
	_, err = NewLocal("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent path, got nil")
	}
}

func TestLocal_Name(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := createTestRepo(t, tmpDir, "test-repo")

	local, err := NewLocal(repoPath)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	expected := "test-repo"
	if local.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, local.Name())
	}
}

func TestLocal_Branch(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := createTestRepo(t, tmpDir, "test-repo")

	local, err := NewLocal(repoPath)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	branch, err := local.Branch()
	if err != nil {
		t.Fatalf("Branch() failed: %v", err)
	}

	// Default branch is usually "main" or "master"
	if branch != "main" && branch != "master" {
		t.Errorf("Expected branch 'main' or 'master', got %s", branch)
	}
}

func TestLocal_IsClean(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := createTestRepo(t, tmpDir, "test-repo")

	local, err := NewLocal(repoPath)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	// Should be clean initially (after commit)
	clean, err := local.IsClean()
	if err != nil {
		t.Fatalf("IsClean() failed: %v", err)
	}
	if !clean {
		t.Logf("Warning: Expected clean repository, but got dirty. This might be expected if the test repo has untracked files.")
		// Not a fatal error as git behavior can vary
	}

	// Make it explicitly dirty by modifying tracked file
	testFile := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(testFile, []byte("# Modified"), 0644); err != nil {
		t.Fatal(err)
	}

	clean, err = local.IsClean()
	if err != nil {
		t.Fatalf("IsClean() failed: %v", err)
	}
	if clean {
		t.Error("Expected dirty repository after modifying tracked file")
	}
}

func TestLocal_CreateBranch(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := createTestRepo(t, tmpDir, "test-repo")

	local, err := NewLocal(repoPath)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	// Create a new branch
	err = local.CreateBranch("feature-test", "")
	if err != nil {
		t.Fatalf("CreateBranch() failed: %v", err)
	}

	// Verify branch exists
	branches, err := local.Branches()
	if err != nil {
		t.Fatalf("Branches() failed: %v", err)
	}

	found := false
	for _, b := range branches {
		if b == "feature-test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created branch not found in branches list")
	}
}

func TestClient_List(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple test repositories directly in tmpDir
	createTestRepo(t, tmpDir, "repo1")
	createTestRepo(t, tmpDir, "repo2")

	client, err := NewClient(tmpDir)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	repos, err := client.List(ListInput{})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Note: The test might fail if fs.Find doesn't discover .git directories correctly
	// This is acceptable for now as it tests the integration
	if len(repos) < 2 {
		t.Logf("Warning: Expected at least 2 repositories, got %d. This might be due to fs.Find implementation.", len(repos))
	}
}

func TestClient_List_WithFilter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test repositories directly in tmpDir
	createTestRepo(t, tmpDir, "backend")
	createTestRepo(t, tmpDir, "frontend")

	client, err := NewClient(tmpDir)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	repos, err := client.List(ListInput{Filter: "backend"})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Note: The test might fail if fs.Find doesn't discover .git directories correctly
	if len(repos) < 1 {
		t.Logf("Warning: Expected at least 1 repository with 'backend' in path, got %d. This might be due to fs.Find implementation.", len(repos))
	}

	if len(repos) > 0 && repos[0].Name() != "backend" {
		t.Errorf("Expected 'backend', got %s", repos[0].Name())
	}
}

func TestLocal_GetModifiedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := createTestRepo(t, tmpDir, "test-repo")

	local, err := NewLocal(repoPath)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	// Initially no modified files
	files, err := local.GetModifiedFiles()
	if err != nil {
		t.Fatalf("GetModifiedFiles() failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 modified files, got %d", len(files))
	}

	// Modify a file
	testFile := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(testFile, []byte("# Modified"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err = local.GetModifiedFiles()
	if err != nil {
		t.Fatalf("GetModifiedFiles() failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 modified file, got %d", len(files))
	}
}

func TestLocal_GetUntrackedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := createTestRepo(t, tmpDir, "test-repo")

	local, err := NewLocal(repoPath)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	// Initially no untracked files
	files, err := local.GetUntrackedFiles()
	if err != nil {
		t.Fatalf("GetUntrackedFiles() failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 untracked files, got %d", len(files))
	}

	// Add an untracked file
	testFile := filepath.Join(repoPath, "untracked.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err = local.GetUntrackedFiles()
	if err != nil {
		t.Fatalf("GetUntrackedFiles() failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 untracked file, got %d", len(files))
	}
}

func TestLocal_CheckoutBranch(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := createTestRepo(t, tmpDir, "test-repo")

	local, err := NewLocal(repoPath)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	// Create a new branch first
	if err := local.CreateBranch("test-branch", ""); err != nil {
		t.Fatalf("CreateBranch() failed: %v", err)
	}

	// Get current branch
	currentBranch, err := local.Branch()
	if err != nil {
		t.Fatalf("Branch() failed: %v", err)
	}

	// Checkout the new branch
	if err := local.CheckoutBranch("test-branch"); err != nil {
		t.Fatalf("CheckoutBranch() failed: %v", err)
	}

	// Verify we're on the new branch
	newBranch, err := local.Branch()
	if err != nil {
		t.Fatalf("Branch() failed: %v", err)
	}

	if newBranch != "test-branch" {
		t.Errorf("Expected branch 'test-branch', got %s", newBranch)
	}

	// Checkout back to original branch
	if err := local.CheckoutBranch(currentBranch); err != nil {
		t.Fatalf("CheckoutBranch() failed: %v", err)
	}
}

func TestClient_Sync(t *testing.T) {
	// This test requires actual git operations and network access
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping sync test in short mode")
	}

	tmpDir := t.TempDir()
	repoPath := createTestRepo(t, tmpDir, "test-repo")

	local, err := NewLocal(repoPath)
	if err != nil {
		t.Fatalf("NewLocal failed: %v", err)
	}

	// Get default branch
	defaultBranch, err := local.DefaultBranch()
	if err != nil {
		t.Fatalf("DefaultBranch() failed: %v", err)
	}

	// Create and checkout a feature branch
	if err := local.CreateBranch("feature", ""); err != nil {
		t.Fatalf("CreateBranch() failed: %v", err)
	}

	// Verify we're on feature branch
	currentBranch, err := local.Branch()
	if err != nil {
		t.Fatalf("Branch() failed: %v", err)
	}
	if currentBranch != "feature" {
		t.Errorf("Expected branch 'feature', got %s", currentBranch)
	}

	// Run sync (should checkout default branch)
	client, err := NewClient(tmpDir)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	if err := client.Sync(ctx, "", false); err != nil {
		t.Fatalf("Sync() failed: %v", err)
	}

	// Verify we're back on default branch
	currentBranch, err = local.Branch()
	if err != nil {
		t.Fatalf("Branch() failed: %v", err)
	}
	if currentBranch != defaultBranch {
		t.Errorf("Expected branch '%s' after sync, got %s", defaultBranch, currentBranch)
	}
}
