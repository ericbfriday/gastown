package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFormatProcessAgeCleanup(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{"seconds", 45, "45s"},
		{"minutes", 125, "2m5s"},
		{"hours", 3665, "1h1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatProcessAgeCleanup(tt.seconds)
			if got != tt.want {
				t.Errorf("formatProcessAgeCleanup(%d) = %s, want %s", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestFindStaleLockFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	runtimeDir := filepath.Join(tmpDir, ".runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a recent lock file (should not be stale)
	recentLock := filepath.Join(runtimeDir, "recent.lock")
	if err := os.WriteFile(recentLock, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an old lock file (should be stale)
	staleLock := filepath.Join(runtimeDir, "stale.lock")
	if err := os.WriteFile(staleLock, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	// Set modification time to 2 hours ago
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(staleLock, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Find stale lock files
	staleFiles, err := findStaleLockFiles(tmpDir)
	if err != nil {
		t.Fatalf("findStaleLockFiles failed: %v", err)
	}

	// Should find exactly one stale lock file
	if len(staleFiles) != 1 {
		t.Errorf("Expected 1 stale lock file, found %d", len(staleFiles))
	}

	if len(staleFiles) > 0 && staleFiles[0] != staleLock {
		t.Errorf("Expected to find %s, got %s", staleLock, staleFiles[0])
	}
}

func TestFindStalePIDFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	runtimeDir := filepath.Join(tmpDir, ".runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create PID file with invalid PID
	invalidPID := filepath.Join(runtimeDir, "invalid.pid")
	if err := os.WriteFile(invalidPID, []byte("not-a-number"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create PID file with non-existent PID
	stalePID := filepath.Join(runtimeDir, "stale.pid")
	if err := os.WriteFile(stalePID, []byte("999999"), 0644); err != nil {
		t.Fatal(err)
	}

	// Find stale PID files
	staleFiles, err := findStalePIDFiles(tmpDir)
	if err != nil {
		t.Fatalf("findStalePIDFiles failed: %v", err)
	}

	// Should find both invalid and stale PID files
	if len(staleFiles) < 1 {
		t.Errorf("Expected at least 1 stale PID file, found %d", len(staleFiles))
	}
}

func TestProcessExists(t *testing.T) {
	// Current process should exist
	if !processExists(os.Getpid()) {
		t.Error("processExists(current PID) should return true")
	}

	// Very high PID unlikely to exist
	if processExists(999999) {
		t.Error("processExists(999999) should return false")
	}
}

func TestGetDirSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	content1 := make([]byte, 1024) // 1KB
	content2 := make([]byte, 2048) // 2KB

	if err := os.WriteFile(file1, content1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, content2, 0644); err != nil {
		t.Fatal(err)
	}

	// Get directory size
	size := getDirSize(tmpDir)

	// Should be 3KB (1KB + 2KB)
	expected := int64(3072)
	if size != expected {
		t.Errorf("getDirSize() = %d, want %d", size, expected)
	}
}

func TestCleanupResultAddDetail(t *testing.T) {
	result := &CleanupResult{}

	result.AddDetail("Test detail %d", 1)
	result.AddDetail("Another detail %s", "test")

	if len(result.Details) != 2 {
		t.Errorf("Expected 2 details, got %d", len(result.Details))
	}

	if result.Details[0] != "Test detail 1" {
		t.Errorf("Unexpected detail: %s", result.Details[0])
	}
}

func TestCleanupResultAddError(t *testing.T) {
	result := &CleanupResult{}

	err1 := os.ErrNotExist
	err2 := os.ErrPermission

	result.AddError(err1)
	result.AddError(err2)

	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(result.Errors))
	}

	if result.Errors[0] != err1 {
		t.Errorf("Unexpected error: %v", result.Errors[0])
	}
}
