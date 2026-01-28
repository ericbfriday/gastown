// Package beads provides tests for package-level wrapper functions.
package beads

import (
	"os"
	"testing"
)

// TestShowPackageFunc tests the package-level Show() function.
func TestShowPackageFunc(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Initialize a beads repo
	b := NewIsolated(tmpDir)
	if err := b.Init("test"); err != nil {
		t.Fatalf("failed to initialize beads repo: %v", err)
	}

	// Create a test issue using the instance method
	issue, err := b.Create(CreateOptions{
		Title:    "Test Issue",
		Type:     "task",
		Priority: 1,
	})
	if err != nil {
		t.Fatalf("failed to create issue: %v", err)
	}

	// Test the package-level Show() function
	retrieved, err := Show(issue.ID)
	if err != nil {
		t.Fatalf("Show() failed: %v", err)
	}

	if retrieved.ID != issue.ID {
		t.Errorf("Show() returned wrong issue: got %s, want %s", retrieved.ID, issue.ID)
	}
	if retrieved.Title != issue.Title {
		t.Errorf("Show() returned wrong title: got %s, want %s", retrieved.Title, issue.Title)
	}
}

// TestListPackageFunc tests the package-level List() function.
func TestListPackageFunc(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Initialize a beads repo
	b := NewIsolated(tmpDir)
	if err := b.Init("test"); err != nil {
		t.Fatalf("failed to initialize beads repo: %v", err)
	}

	// Create test issues using the instance method
	task1, err := b.Create(CreateOptions{
		Title:    "Task 1",
		Type:     "task",
		Priority: 1,
	})
	if err != nil {
		t.Fatalf("failed to create task1: %v", err)
	}

	task2, err := b.Create(CreateOptions{
		Title:    "Task 2",
		Type:     "task",
		Priority: 2,
	})
	if err != nil {
		t.Fatalf("failed to create task2: %v", err)
	}

	// Test the package-level List() function
	issues, err := List(ListOptions{
		Status:   "all",
		Priority: -1, // No priority filter
	})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Verify we got both issues
	if len(issues) != 2 {
		t.Errorf("List() returned wrong count: got %d, want 2", len(issues))
	}

	// Verify the issues are in the result (order may vary)
	found := make(map[string]bool)
	for _, issue := range issues {
		found[issue.ID] = true
	}

	if !found[task1.ID] {
		t.Errorf("List() missing task1: %s", task1.ID)
	}
	if !found[task2.ID] {
		t.Errorf("List() missing task2: %s", task2.ID)
	}

	// Test filtering by priority
	highPriority, err := List(ListOptions{
		Status:   "all",
		Priority: 2,
	})
	if err != nil {
		t.Fatalf("List() with priority filter failed: %v", err)
	}

	if len(highPriority) != 1 {
		t.Errorf("List() with priority=2 returned wrong count: got %d, want 1", len(highPriority))
	}
	if len(highPriority) > 0 && highPriority[0].ID != task2.ID {
		t.Errorf("List() with priority=2 returned wrong issue: got %s, want %s",
			highPriority[0].ID, task2.ID)
	}
}

// TestShowNotFound tests Show() with non-existent ID.
func TestShowNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Initialize a beads repo
	b := NewIsolated(tmpDir)
	if err := b.Init("test"); err != nil {
		t.Fatalf("failed to initialize beads repo: %v", err)
	}

	// Try to show non-existent issue
	_, err = Show("test-nonexistent")
	if err != ErrNotFound {
		t.Errorf("Show() with non-existent ID should return ErrNotFound, got: %v", err)
	}
}

// TestListReturnType verifies List() returns []Issue not []*Issue.
func TestListReturnType(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	b := NewIsolated(tmpDir)
	if err := b.Init("test"); err != nil {
		t.Fatalf("failed to initialize beads repo: %v", err)
	}

	_, err = b.Create(CreateOptions{
		Title:    "Test",
		Type:     "task",
		Priority: 1,
	})
	if err != nil {
		t.Fatalf("failed to create issue: %v", err)
	}

	// Get issues
	issues, err := List(ListOptions{Status: "all", Priority: -1})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Verify we can take address of slice elements (as planoracle does)
	if len(issues) > 0 {
		ptr := &issues[0]
		if ptr.ID == "" {
			t.Error("taking address of issue should work")
		}
	}
}
