package beads

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTemplate(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	templatesDir := filepath.Join(tmpDir, ".beads", "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Write a test template
	templateContent := `
name = "test-template"
description = "Test template"
version = 1

[vars.feature]
description = "Feature name"
required = true

[vars.priority]
description = "Priority"
default = "2"

[epic]
title = "Test {{feature}}"
description = "Testing {{feature}}"
priority = "{{priority}}"
type = "epic"

[[subtasks]]
title = "Subtask for {{feature}}"
description = "Do work"
type = "task"
priority = "{{priority}}"
`
	templatePath := filepath.Join(templatesDir, "test-template.template.toml")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write test template: %v", err)
	}

	// Set BEADS_DIR for test
	oldBeadsDir := os.Getenv("BEADS_DIR")
	os.Setenv("BEADS_DIR", filepath.Join(tmpDir, ".beads"))
	defer os.Setenv("BEADS_DIR", oldBeadsDir)

	// Test loading template
	tmpl, err := LoadTemplate("test-template")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if tmpl.Name != "test-template" {
		t.Errorf("Expected name 'test-template', got '%s'", tmpl.Name)
	}

	if tmpl.Epic.Title != "Test {{feature}}" {
		t.Errorf("Expected epic title 'Test {{feature}}', got '%s'", tmpl.Epic.Title)
	}

	if len(tmpl.Subtasks) != 1 {
		t.Errorf("Expected 1 subtask, got %d", len(tmpl.Subtasks))
	}

	// Test loading non-existent template
	_, err = LoadTemplate("nonexistent")
	if err == nil {
		t.Error("Expected error loading non-existent template")
	}
}

func TestListTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	templatesDir := filepath.Join(tmpDir, ".beads", "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create some test templates
	templates := []string{"template1", "template2", "template3"}
	for _, name := range templates {
		path := filepath.Join(templatesDir, name+".template.toml")
		if err := os.WriteFile(path, []byte("name = \""+name+"\""), 0644); err != nil {
			t.Fatalf("Failed to write template %s: %v", name, err)
		}
	}

	// Create a non-template file (should be ignored)
	otherFile := filepath.Join(templatesDir, "README.md")
	if err := os.WriteFile(otherFile, []byte("# Templates"), 0644); err != nil {
		t.Fatalf("Failed to write other file: %v", err)
	}

	oldBeadsDir := os.Getenv("BEADS_DIR")
	os.Setenv("BEADS_DIR", filepath.Join(tmpDir, ".beads"))
	defer os.Setenv("BEADS_DIR", oldBeadsDir)

	found, err := ListTemplates()
	if err != nil {
		t.Fatalf("Failed to list templates: %v", err)
	}

	if len(found) != 3 {
		t.Errorf("Expected 3 templates, got %d: %v", len(found), found)
	}

	// Verify all expected templates are present
	foundMap := make(map[string]bool)
	for _, name := range found {
		foundMap[name] = true
	}

	for _, expected := range templates {
		if !foundMap[expected] {
			t.Errorf("Expected to find template '%s'", expected)
		}
	}
}

func TestExpandTemplate(t *testing.T) {
	tmpl := &EpicTemplate{
		Name: "test",
		Vars: map[string]TemplateVar{
			"feature": {
				Description: "Feature name",
				Required:    true,
			},
			"priority": {
				Description: "Priority",
				Default:     "2",
			},
		},
		Epic: EpicDefinition{
			Title:       "Add {{feature}}",
			Description: "Implement {{feature}} with priority {{priority}}",
			Priority:    "{{priority}}",
			Type:        "epic",
		},
		Subtasks: []SubtaskDefinition{
			{
				Title:       "Design {{feature}}",
				Description: "Design the {{feature}} implementation",
				Type:        "task",
				Priority:    "{{priority}}",
			},
			{
				Title:       "Implement {{feature}}",
				Description: "Build {{feature}}",
				Type:        "task",
				Priority:    "1",
			},
		},
	}

	vars := map[string]string{
		"feature": "logging",
	}

	expanded, err := ExpandTemplate(tmpl, vars)
	if err != nil {
		t.Fatalf("Failed to expand template: %v", err)
	}

	// Check epic
	if expanded.Epic.Title != "Add logging" {
		t.Errorf("Expected epic title 'Add logging', got '%s'", expanded.Epic.Title)
	}

	if expanded.Epic.Description != "Implement logging with priority 2" {
		t.Errorf("Unexpected epic description: %s", expanded.Epic.Description)
	}

	if expanded.Epic.Priority != 2 {
		t.Errorf("Expected priority 2, got %d", expanded.Epic.Priority)
	}

	// Check subtasks
	if len(expanded.Subtasks) != 2 {
		t.Fatalf("Expected 2 subtasks, got %d", len(expanded.Subtasks))
	}

	if expanded.Subtasks[0].Title != "Design logging" {
		t.Errorf("Expected subtask title 'Design logging', got '%s'", expanded.Subtasks[0].Title)
	}

	if expanded.Subtasks[1].Priority != 1 {
		t.Errorf("Expected subtask priority 1, got %d", expanded.Subtasks[1].Priority)
	}

	// Test missing required variable
	_, err = ExpandTemplate(tmpl, map[string]string{})
	if err == nil {
		t.Error("Expected error for missing required variable")
	}
}

func TestExpandTemplateWithList(t *testing.T) {
	tmpl := &EpicTemplate{
		Name: "cross-rig",
		Vars: map[string]TemplateVar{
			"feature": {
				Description: "Feature name",
				Required:    true,
			},
			"rigs": {
				Description: "Comma-separated rig list",
				Required:    true,
			},
		},
		Epic: EpicDefinition{
			Title:       "Add {{feature}} to all rigs",
			Description: "Cross-rig: {{feature}}",
			Type:        "epic",
		},
		Subtasks: []SubtaskDefinition{
			{
				Title:       "{{rig}}: {{feature}}",
				Description: "Add {{feature}} to {{rig}}",
				Type:        "task",
				ExpandOver:  "rigs",
			},
		},
	}

	vars := map[string]string{
		"feature": "metrics",
		"rigs":    "gastown, beads, aardwolf_snd",
	}

	expanded, err := ExpandTemplate(tmpl, vars)
	if err != nil {
		t.Fatalf("Failed to expand template: %v", err)
	}

	// Should create one subtask per rig
	if len(expanded.Subtasks) != 3 {
		t.Fatalf("Expected 3 subtasks (one per rig), got %d", len(expanded.Subtasks))
	}

	expectedTitles := []string{
		"gastown: metrics",
		"beads: metrics",
		"aardwolf_snd: metrics",
	}

	for i, expected := range expectedTitles {
		if expanded.Subtasks[i].Title != expected {
			t.Errorf("Expected subtask %d title '%s', got '%s'", i, expected, expanded.Subtasks[i].Title)
		}
	}

	// Verify descriptions
	if expanded.Subtasks[0].Description != "Add metrics to gastown" {
		t.Errorf("Unexpected description: %s", expanded.Subtasks[0].Description)
	}
}

func TestExpandString(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple substitution",
			template: "Hello {{name}}",
			vars:     map[string]string{"name": "world"},
			expected: "Hello world",
		},
		{
			name:     "multiple vars",
			template: "{{greeting}} {{name}}",
			vars:     map[string]string{"greeting": "Hi", "name": "Claude"},
			expected: "Hi Claude",
		},
		{
			name:     "no variables",
			template: "Static text",
			vars:     map[string]string{},
			expected: "Static text",
		},
		{
			name:     "missing variable",
			template: "Hello {{name}}",
			vars:     map[string]string{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandString(tt.template, tt.vars)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
