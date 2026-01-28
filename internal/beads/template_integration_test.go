package beads

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTemplateIntegration tests the full template workflow
func TestTemplateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary test directory
	tmpDir := t.TempDir()
	templatesDir := filepath.Join(tmpDir, ".beads", "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create a simple test template
	templateContent := `
name = "test-integration"
description = "Integration test template"
version = 1

[vars.project]
description = "Project name"
required = true

[vars.tasks]
description = "Comma-separated task list"
required = true

[vars.priority]
description = "Priority"
default = "2"

[epic]
title = "Complete {{project}}"
description = "Work items for {{project}}"
priority = "{{priority}}"
type = "epic"

[[subtasks]]
title = "{{project}}: {{task}}"
description = "Do {{task}} for {{project}}"
type = "task"
priority = "{{priority}}"
expand_over = "tasks"
`

	templatePath := filepath.Join(templatesDir, "test-integration.template.toml")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write test template: %v", err)
	}

	// Set BEADS_DIR for test
	oldBeadsDir := os.Getenv("BEADS_DIR")
	os.Setenv("BEADS_DIR", filepath.Join(tmpDir, ".beads"))
	defer os.Setenv("BEADS_DIR", oldBeadsDir)

	// Test 1: List templates
	templates, err := ListTemplates()
	if err != nil {
		t.Fatalf("Failed to list templates: %v", err)
	}

	if len(templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(templates))
	}

	if templates[0] != "test-integration" {
		t.Errorf("Expected template 'test-integration', got '%s'", templates[0])
	}

	// Test 2: Load template
	tmpl, err := LoadTemplate("test-integration")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if tmpl.Name != "test-integration" {
		t.Errorf("Expected name 'test-integration', got '%s'", tmpl.Name)
	}

	// Test 3: Expand template
	vars := map[string]string{
		"project": "Beads",
		"tasks":   "design,implement,test",
	}

	expanded, err := ExpandTemplate(tmpl, vars)
	if err != nil {
		t.Fatalf("Failed to expand template: %v", err)
	}

	// Verify epic
	if expanded.Epic.Title != "Complete Beads" {
		t.Errorf("Expected epic title 'Complete Beads', got '%s'", expanded.Epic.Title)
	}

	// Verify subtasks
	if len(expanded.Subtasks) != 3 {
		t.Fatalf("Expected 3 subtasks, got %d", len(expanded.Subtasks))
	}

	expectedSubtasks := []string{
		"Beads: design",
		"Beads: implement",
		"Beads: test",
	}

	for i, expected := range expectedSubtasks {
		if expanded.Subtasks[i].Title != expected {
			t.Errorf("Expected subtask %d title '%s', got '%s'", i, expected, expanded.Subtasks[i].Title)
		}
	}

	// Verify all subtasks have correct priority
	for i, subtask := range expanded.Subtasks {
		if subtask.Priority != 2 {
			t.Errorf("Expected subtask %d priority 2, got %d", i, subtask.Priority)
		}
	}
}
