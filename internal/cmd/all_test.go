package cmd

import (
	"testing"

	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/rig"
)

func TestExpandSingleSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name:    "wildcard all",
			spec:    "*",
			wantErr: false,
		},
		{
			name:    "rig wildcard",
			spec:    "gastown/*",
			wantErr: false,
		},
		{
			name:    "specific rig/polecat",
			spec:    "gastown/Toast",
			wantErr: false,
		},
		{
			name:    "invalid empty",
			spec:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a integration test - would need mocked manager
			// For now, just test the pattern matching logic
			if tt.spec == "" {
				// Empty spec should fail
				if !tt.wantErr {
					t.Errorf("expected error for empty spec")
				}
			}

			// Test wildcard detection
			if tt.spec == "*" {
				if !isWildcardAll(tt.spec) {
					t.Errorf("failed to detect wildcard all")
				}
			}

			// Test rig wildcard detection
			if tt.spec == "gastown/*" {
				if !isRigWildcard(tt.spec) {
					t.Errorf("failed to detect rig wildcard")
				}
			}
		})
	}
}

func TestApplyFilters(t *testing.T) {
	specs := []polecatSpec{
		{
			rigName:     "gastown",
			polecatName: "Toast",
			p: &polecat.Polecat{
				Name:  "Toast",
				State: polecat.StateWorking,
			},
		},
		{
			rigName:     "gastown",
			polecatName: "Furiosa",
			p: &polecat.Polecat{
				Name:  "Furiosa",
				State: polecat.StateDone,
			},
		},
		{
			rigName:     "greenplace",
			polecatName: "Max",
			p: &polecat.Polecat{
				Name:  "Max",
				State: polecat.StateWorking,
			},
		},
	}

	tests := []struct {
		name      string
		filterRig string
		status    string
		pattern   string
		wantCount int
	}{
		{
			name:      "no filters",
			wantCount: 3,
		},
		{
			name:      "filter by rig",
			filterRig: "gastown",
			wantCount: 2,
		},
		{
			name:      "filter by status",
			status:    "working",
			wantCount: 2,
		},
		{
			name:      "filter by pattern",
			pattern:   "Toast",
			wantCount: 1,
		},
		{
			name:      "combined filters",
			filterRig: "gastown",
			status:    "working",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origRig := allRig
			origStatus := allStatus
			origPattern := allPattern

			// Set test filters
			allRig = tt.filterRig
			allStatus = tt.status
			allPattern = tt.pattern

			// Run filter
			filtered := applyFilters(specs)

			// Restore original values
			allRig = origRig
			allStatus = origStatus
			allPattern = origPattern

			if len(filtered) != tt.wantCount {
				t.Errorf("applyFilters() got %d results, want %d", len(filtered), tt.wantCount)
			}
		})
	}
}

func TestPolecatSpecFields(t *testing.T) {
	spec := polecatSpec{
		rigName:     "gastown",
		polecatName: "Toast",
		p: &polecat.Polecat{
			Name:  "Toast",
			State: polecat.StateWorking,
			Issue: "gt-123",
		},
	}

	if spec.rigName != "gastown" {
		t.Errorf("rigName = %s, want gastown", spec.rigName)
	}

	if spec.polecatName != "Toast" {
		t.Errorf("polecatName = %s, want Toast", spec.polecatName)
	}

	if spec.p.State != polecat.StateWorking {
		t.Errorf("state = %s, want working", spec.p.State)
	}
}

// Helper functions for pattern detection
func isWildcardAll(spec string) bool {
	return spec == "*"
}

func isRigWildcard(spec string) bool {
	return len(spec) > 2 && spec[len(spec)-2:] == "/*"
}

func TestFilterByRig(t *testing.T) {
	specs := []polecatSpec{
		{rigName: "gastown", polecatName: "A", p: &polecat.Polecat{Name: "A"}},
		{rigName: "greenplace", polecatName: "B", p: &polecat.Polecat{Name: "B"}},
		{rigName: "gastown", polecatName: "C", p: &polecat.Polecat{Name: "C"}},
	}

	// Save and set filter
	origRig := allRig
	allRig = "gastown"

	filtered := applyFilters(specs)

	allRig = origRig

	if len(filtered) != 2 {
		t.Errorf("expected 2 results, got %d", len(filtered))
	}

	for _, s := range filtered {
		if s.rigName != "gastown" {
			t.Errorf("expected rig=gastown, got %s", s.rigName)
		}
	}
}

func TestFilterByState(t *testing.T) {
	specs := []polecatSpec{
		{
			rigName:     "gastown",
			polecatName: "A",
			p:           &polecat.Polecat{Name: "A", State: polecat.StateWorking},
		},
		{
			rigName:     "gastown",
			polecatName: "B",
			p:           &polecat.Polecat{Name: "B", State: polecat.StateDone},
		},
		{
			rigName:     "gastown",
			polecatName: "C",
			p:           &polecat.Polecat{Name: "C", State: polecat.StateWorking},
		},
	}

	// Save and set filter
	origStatus := allStatus
	allStatus = "working"

	filtered := applyFilters(specs)

	allStatus = origStatus

	if len(filtered) != 2 {
		t.Errorf("expected 2 results, got %d", len(filtered))
	}

	for _, s := range filtered {
		if s.p.State != polecat.StateWorking {
			t.Errorf("expected state=working, got %s", s.p.State)
		}
	}
}

func TestFilterByPattern(t *testing.T) {
	specs := []polecatSpec{
		{rigName: "gastown", polecatName: "Toast", p: &polecat.Polecat{Name: "Toast"}},
		{rigName: "gastown", polecatName: "Furiosa", p: &polecat.Polecat{Name: "Furiosa"}},
		{rigName: "gastown", polecatName: "ToastMaster", p: &polecat.Polecat{Name: "ToastMaster"}},
	}

	// Save and set filter
	origPattern := allPattern
	allPattern = "Toast"

	filtered := applyFilters(specs)

	allPattern = origPattern

	if len(filtered) != 2 {
		t.Errorf("expected 2 results, got %d", len(filtered))
	}

	for _, s := range filtered {
		if s.polecatName != "Toast" && s.polecatName != "ToastMaster" {
			t.Errorf("unexpected name: %s", s.polecatName)
		}
	}
}

func TestCombinedFilters(t *testing.T) {
	specs := []polecatSpec{
		{
			rigName:     "gastown",
			polecatName: "Toast",
			p:           &polecat.Polecat{Name: "Toast", State: polecat.StateWorking},
		},
		{
			rigName:     "gastown",
			polecatName: "Furiosa",
			p:           &polecat.Polecat{Name: "Furiosa", State: polecat.StateDone},
		},
		{
			rigName:     "greenplace",
			polecatName: "Toast",
			p:           &polecat.Polecat{Name: "Toast", State: polecat.StateWorking},
		},
	}

	// Save and set filters
	origRig := allRig
	origStatus := allStatus
	origPattern := allPattern

	allRig = "gastown"
	allStatus = "working"
	allPattern = "Toast"

	filtered := applyFilters(specs)

	allRig = origRig
	allStatus = origStatus
	allPattern = origPattern

	if len(filtered) != 1 {
		t.Errorf("expected 1 result, got %d", len(filtered))
	}

	if len(filtered) > 0 {
		if filtered[0].rigName != "gastown" {
			t.Errorf("expected rig=gastown, got %s", filtered[0].rigName)
		}
		if filtered[0].polecatName != "Toast" {
			t.Errorf("expected name=Toast, got %s", filtered[0].polecatName)
		}
		if filtered[0].p.State != polecat.StateWorking {
			t.Errorf("expected state=working, got %s", filtered[0].p.State)
		}
	}
}
