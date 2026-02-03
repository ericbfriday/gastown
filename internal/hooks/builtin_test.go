package hooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckCommitsPushed(t *testing.T) {
	// Create a temporary git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	configEmail := exec.Command("git", "config", "user.email", "test@example.com")
	configEmail.Dir = tmpDir
	configEmail.Run()

	configName := exec.Command("git", "config", "user.name", "Test User")
	configName.Dir = tmpDir
	configName.Run()

	ctx := &HookContext{
		WorkingDir: tmpDir,
	}

	// Test 1: Empty repo - should pass (can't get current branch yet)
	result, err := checkCommitsPushed(ctx)
	if err != nil {
		t.Errorf("checkCommitsPushed failed: %v", err)
	}
	// Empty repo will fail to get branch, which returns non-blocking success
	if !result.Success {
		t.Logf("Expected success or acceptable failure for empty repo: %s", result.Message)
	}

	// Create a commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = tmpDir
	addCmd.Run()

	commitCmd := exec.Command("git", "commit", "-m", "test commit")
	commitCmd.Dir = tmpDir
	commitCmd.Run()

	// Test 2: No upstream branch - should pass with message
	result, err = checkCommitsPushed(ctx)
	if err != nil {
		t.Errorf("checkCommitsPushed failed: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success for branch without upstream: %s", result.Message)
	}
}

func TestCheckBeadsSynced(t *testing.T) {
	ctx := &HookContext{
		WorkingDir: t.TempDir(),
	}

	// Test: bd sync not available - should pass
	result, err := checkBeadsSynced(ctx)
	if err != nil {
		t.Errorf("checkBeadsSynced failed: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success when bd sync not available: %s", result.Message)
	}
}

func TestCheckAssignedIssues(t *testing.T) {
	ctx := &HookContext{
		WorkingDir: t.TempDir(),
		Metadata:   map[string]interface{}{},
	}

	// Test 1: No polecat metadata - should pass
	result, err := checkAssignedIssues(ctx)
	if err != nil {
		t.Errorf("checkAssignedIssues failed: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success with no polecat metadata")
	}

	// Test 2: With polecat metadata but bd not available - should pass
	ctx.Metadata["polecat"] = "test-polecat"
	ctx.Metadata["rig"] = "test-rig"
	result, err = checkAssignedIssues(ctx)
	if err != nil {
		t.Errorf("checkAssignedIssues failed: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success when bd not available: %s", result.Message)
	}
}

func TestVerifyGitClean(t *testing.T) {
	// Create a temporary git repo
	tmpDir := t.TempDir()

	ctx := &HookContext{
		WorkingDir: tmpDir,
	}

	// Test 1: Not a git repo - should pass
	result, err := verifyGitClean(ctx)
	if err != nil {
		t.Errorf("verifyGitClean failed: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success for non-git directory")
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Test 2: Clean repo - should pass
	result, err = verifyGitClean(ctx)
	if err != nil {
		t.Errorf("verifyGitClean failed: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success for clean repo: %s", result.Message)
	}

	// Create an untracked file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test 3: Uncommitted changes - should fail and block
	result, err = verifyGitClean(ctx)
	if err != nil {
		t.Errorf("verifyGitClean failed: %v", err)
	}
	if result.Success {
		t.Errorf("Expected failure for dirty repo")
	}
	if !result.Block {
		t.Errorf("Expected Block=true for uncommitted changes")
	}
}

func TestPreShutdownChecks(t *testing.T) {
	// Create a clean temporary git repo
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	ctx := &HookContext{
		WorkingDir: tmpDir,
		Metadata: map[string]interface{}{
			"polecat": "test-polecat",
			"rig":     "test-rig",
		},
	}

	// Test: Clean state - should pass all checks
	result, err := preShutdownChecks(ctx)
	if err != nil {
		t.Errorf("preShutdownChecks failed: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected success for clean state: %s", result.Message)
	}

	// Create uncommitted changes
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test: Dirty state - should fail and block
	result, err = preShutdownChecks(ctx)
	if err != nil {
		t.Errorf("preShutdownChecks failed: %v", err)
	}
	if result.Success {
		t.Errorf("Expected failure for dirty state")
	}
	if !result.Block {
		t.Errorf("Expected Block=true for dirty state")
	}
}

func TestFindGitDir(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	subDir := filepath.Join(tmpDir, "subdir")

	// Test 1: No .git directory
	result := findGitDir(tmpDir)
	if result != "" {
		t.Errorf("Expected empty string for directory without .git, got: %s", result)
	}

	// Create .git directory
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Test 2: .git exists
	result = findGitDir(tmpDir)
	if result != gitDir {
		t.Errorf("Expected %s, got: %s", gitDir, result)
	}

	// Create subdirectory
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Test 3: Search from subdirectory should find parent's .git
	result = findGitDir(subDir)
	if result != gitDir {
		t.Errorf("Expected %s, got: %s", gitDir, result)
	}
}

func TestJoinMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
		expected string
	}{
		{
			name:     "empty",
			messages: []string{},
			expected: "",
		},
		{
			name:     "single message",
			messages: []string{"error 1"},
			expected: "  - error 1",
		},
		{
			name:     "multiple messages",
			messages: []string{"error 1", "error 2", "error 3"},
			expected: "  - error 1\n  - error 2\n  - error 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinMessages(tt.messages)
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}
