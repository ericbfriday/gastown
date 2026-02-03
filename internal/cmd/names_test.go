package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/gastown/internal/polecat"
)

func TestValidateNameFormat(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid lowercase", "furiosa", false},
		{"valid uppercase", "FURIOSA", false},
		{"valid with hyphen", "mad-max", false},
		{"valid with underscore", "mad_max", false},
		{"valid alphanumeric", "polecat01", false},
		{"empty string", "", true},
		{"with spaces", "mad max", true},
		{"with special chars", "mad@max", true},
		{"too long", "thisisaverylongnamethatexceedsthirtytwocharacterslimit", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNameFormat(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("validateNameFormat(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestNamePoolOperations(t *testing.T) {
	// Create temp rig directory
	tmpDir := t.TempDir()
	rigName := "testrig"
	rigPath := filepath.Join(tmpDir, rigName)

	if err := os.MkdirAll(rigPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a name pool
	pool := polecat.NewNamePool(rigPath, rigName)

	// Test initial state
	if pool.ActiveCount() != 0 {
		t.Errorf("new pool should have 0 active names, got %d", pool.ActiveCount())
	}

	// Test allocation
	name1, err := pool.Allocate()
	if err != nil {
		t.Fatalf("failed to allocate name: %v", err)
	}
	if name1 == "" {
		t.Fatal("allocated name should not be empty")
	}

	// Test that allocated name is marked in use
	if pool.ActiveCount() != 1 {
		t.Errorf("pool should have 1 active name, got %d", pool.ActiveCount())
	}

	// Test adding custom name
	customName := "testname"
	pool.AddCustomName(customName)
	if !pool.IsCustomName(customName) {
		t.Errorf("name %q should be a custom name", customName)
	}

	// Test HasName
	if !pool.HasName(name1) {
		t.Errorf("pool should have allocated name %q", name1)
	}
	if !pool.HasName(customName) {
		t.Errorf("pool should have custom name %q", customName)
	}

	// Test IsInUse
	if !pool.IsInUse(name1) {
		t.Errorf("name %q should be in use", name1)
	}
	if pool.IsInUse(customName) {
		t.Errorf("custom name %q should not be in use", customName)
	}

	// Test release
	pool.Release(name1)
	if pool.IsInUse(name1) {
		t.Errorf("name %q should not be in use after release", name1)
	}

	// Test removing custom name
	if err := pool.RemoveCustomName(customName); err != nil {
		t.Errorf("failed to remove custom name: %v", err)
	}
	if pool.HasName(customName) {
		t.Errorf("pool should not have custom name %q after removal", customName)
	}
}

func TestPercentage(t *testing.T) {
	tests := []struct {
		part  int
		total int
		want  int
	}{
		{0, 100, 0},
		{50, 100, 50},
		{100, 100, 100},
		{25, 100, 25},
		{1, 3, 33},
		{0, 0, 0}, // edge case: division by zero
	}

	for _, tt := range tests {
		got := percentage(tt.part, tt.total)
		if got != tt.want {
			t.Errorf("percentage(%d, %d) = %d, want %d", tt.part, tt.total, got, tt.want)
		}
	}
}

func TestNamePoolStats(t *testing.T) {
	tmpDir := t.TempDir()
	rigName := "testrig"
	rigPath := filepath.Join(tmpDir, rigName)

	if err := os.MkdirAll(rigPath, 0755); err != nil {
		t.Fatal(err)
	}

	pool := polecat.NewNamePool(rigPath, rigName)

	// Add some custom names
	pool.AddCustomName("custom1")
	pool.AddCustomName("custom2")

	// Allocate a few names
	_, _ = pool.Allocate()
	_, _ = pool.Allocate()

	// Test stats
	if pool.CustomNameCount() != 2 {
		t.Errorf("expected 2 custom names, got %d", pool.CustomNameCount())
	}

	if pool.ActiveCount() != 2 {
		t.Errorf("expected 2 active names, got %d", pool.ActiveCount())
	}

	totalNames := pool.TotalNames()
	if totalNames == 0 {
		t.Error("total names should not be zero")
	}
}

func TestListExistingPolecats(t *testing.T) {
	tmpDir := t.TempDir()
	rigPath := filepath.Join(tmpDir, "testrig")
	polecatsDir := filepath.Join(rigPath, "polecats")

	// Create polecats directory
	if err := os.MkdirAll(polecatsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some polecat directories
	polecatNames := []string{"furiosa", "nux", "capable"}
	for _, name := range polecatNames {
		polecatPath := filepath.Join(polecatsDir, name)
		if err := os.MkdirAll(polecatPath, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create a hidden directory (should be ignored)
	hiddenPath := filepath.Join(polecatsDir, ".hidden")
	if err := os.MkdirAll(hiddenPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Test listing
	existing := listExistingPolecats(rigPath)

	if len(existing) != len(polecatNames) {
		t.Errorf("expected %d polecats, got %d", len(polecatNames), len(existing))
	}

	// Check that all expected names are present
	nameMap := make(map[string]bool)
	for _, name := range existing {
		nameMap[name] = true
	}

	for _, expected := range polecatNames {
		if !nameMap[expected] {
			t.Errorf("expected polecat %q not found in list", expected)
		}
	}

	// Check that hidden directory is not included
	if nameMap[".hidden"] {
		t.Error("hidden directory should not be in list")
	}
}
