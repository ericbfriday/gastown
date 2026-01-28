package planconvert

import (
	"path/filepath"
	"testing"
)

func TestParsePlanDocument(t *testing.T) {
	testFile := "../../testdata/plans/example-phases.md"

	doc, err := ParsePlanDocument(testFile)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Verify document title
	if doc.Title != "Example Multi-Phase Project" {
		t.Errorf("Expected title 'Example Multi-Phase Project', got '%s'", doc.Title)
	}

	// Verify metadata
	if doc.Metadata.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", doc.Metadata.Version)
	}

	if doc.Metadata.Status != "Draft" {
		t.Errorf("Expected status 'Draft', got '%s'", doc.Metadata.Status)
	}

	// Verify sections were parsed
	if len(doc.Sections) == 0 {
		t.Errorf("Expected sections to be parsed, got 0")
	}

	t.Logf("Parsed document: %s", doc.Title)
	t.Logf("Sections found: %d", len(doc.Sections))
	for i, section := range doc.Sections {
		t.Logf("  Section %d: %s (type: %s)", i+1, section.Title, section.Type)
	}
}

func TestConvertToEpic(t *testing.T) {
	testFile := filepath.Join("../../testdata/plans/example-phases.md")

	doc, err := ParsePlanDocument(testFile)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	opts := ConversionOptions{
		Prefix:   "test",
		Priority: 2,
	}

	epic, err := ConvertToEpic(doc, opts)
	if err != nil {
		t.Fatalf("Failed to convert to epic: %v", err)
	}

	// Verify epic was created
	if epic.Title != doc.Title {
		t.Errorf("Expected epic title '%s', got '%s'", doc.Title, epic.Title)
	}

	// Verify subtasks were created
	if len(epic.Subtasks) == 0 {
		t.Errorf("Expected subtasks to be created, got 0")
	}

	t.Logf("Epic created: %s", epic.Title)
	t.Logf("Subtasks: %d", len(epic.Subtasks))
	for i, task := range epic.Subtasks {
		t.Logf("  Task %d: [%s] %s (phase: %s)", i+1, task.ID, task.Title, task.Phase)
	}
}
