package planconvert

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParseYAMLFrontmatter(t *testing.T) {
	// Create temporary test file with YAML frontmatter
	content := `---
version: 2.0
status: active
date: 2026-01-28
author: Test Author
phase: Implementation
---

# Test Document

This is a test document with YAML frontmatter.
`

	tmpFile := filepath.Join(t.TempDir(), "test-yaml.md")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	doc, err := ParsePlanDocument(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Verify metadata was extracted from YAML frontmatter
	if doc.Metadata.Version != "2.0" {
		t.Errorf("Expected version '2.0', got '%s'", doc.Metadata.Version)
	}

	if doc.Metadata.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", doc.Metadata.Status)
	}

	if doc.Metadata.Date != "2026-01-28" {
		t.Errorf("Expected date '2026-01-28', got '%s'", doc.Metadata.Date)
	}

	if doc.Metadata.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", doc.Metadata.Author)
	}

	if doc.Metadata.Phase != "Implementation" {
		t.Errorf("Expected phase 'Implementation', got '%s'", doc.Metadata.Phase)
	}

	if doc.Title != "Test Document" {
		t.Errorf("Expected title 'Test Document', got '%s'", doc.Title)
	}
}

func TestParseCheckboxTasks(t *testing.T) {
	content := `# Test Checkbox Tasks

## Task Section

- [ ] First unchecked task
- [x] Completed task
- [ ] Third task
- [ ] Fourth task with details
`

	tmpFile := filepath.Join(t.TempDir(), "test-checkboxes.md")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	doc, err := ParsePlanDocument(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Extract tasks from sections
	var allTasks []Task
	for _, section := range doc.Sections {
		tasks := ExtractTasks(&section, section.Title)
		allTasks = append(allTasks, tasks...)
	}

	if len(allTasks) == 0 {
		t.Fatalf("Expected tasks to be extracted, got 0")
	}

	// Verify tasks were extracted
	expectedTitles := []string{
		"First unchecked task",
		"Completed task",
		"Third task",
		"Fourth task with details",
	}

	if len(allTasks) != len(expectedTitles) {
		t.Errorf("Expected %d tasks, got %d", len(expectedTitles), len(allTasks))
	}

	for i, task := range allTasks {
		if i < len(expectedTitles) && task.Title != expectedTitles[i] {
			t.Errorf("Task %d: expected title '%s', got '%s'", i, expectedTitles[i], task.Title)
		}
	}

	t.Logf("Extracted %d checkbox tasks", len(allTasks))
	for i, task := range allTasks {
		t.Logf("  Task %d: %s", i+1, task.Title)
	}
}

func TestParseEmojiCheckboxTasks(t *testing.T) {
	content := `# Test Emoji Tasks

## Deliverables

- ✅ Completed deliverable
- ☐ Pending deliverable
- ✅ Another completed item
`

	tmpFile := filepath.Join(t.TempDir(), "test-emoji.md")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	doc, err := ParsePlanDocument(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Extract tasks
	var allTasks []Task
	for _, section := range doc.Sections {
		tasks := ExtractTasks(&section, section.Title)
		allTasks = append(allTasks, tasks...)
	}

	if len(allTasks) == 0 {
		t.Fatalf("Expected emoji checkbox tasks to be extracted, got 0")
	}

	t.Logf("Extracted %d emoji checkbox tasks", len(allTasks))
	for i, task := range allTasks {
		t.Logf("  Task %d: %s", i+1, task.Title)
	}
}

func TestParseMixedTaskFormats(t *testing.T) {
	content := `# Mixed Task Formats

## Phase 1: Setup

**Tasks:**
1. Configure environment
2. Install dependencies
3. Setup database

**Deliverables:**
- ✅ Environment configured
- ✅ Dependencies installed

## Phase 2: Implementation

- [ ] Implement feature A
- [ ] Implement feature B
- [x] Setup completed
`

	tmpFile := filepath.Join(t.TempDir(), "test-mixed.md")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	doc, err := ParsePlanDocument(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Extract tasks from all sections
	var allTasks []Task
	for _, section := range doc.Sections {
		tasks := ExtractTasks(&section, section.Title)
		allTasks = append(allTasks, tasks...)

		// Process subsections
		for _, subsection := range section.Subsections {
			subTasks := ExtractTasks(&subsection, subsection.Title)
			allTasks = append(allTasks, subTasks...)
		}
	}

	if len(allTasks) == 0 {
		t.Fatalf("Expected tasks to be extracted from mixed formats, got 0")
	}

	t.Logf("Extracted %d tasks from mixed formats", len(allTasks))
	for i, task := range allTasks {
		t.Logf("  Task %d: %s (phase: %s)", i+1, task.Title, task.Phase)
		if len(task.Deliverables) > 0 {
			t.Logf("    Deliverables: %d", len(task.Deliverables))
		}
	}

	// Verify we extracted both numbered and checkbox tasks
	hasNumberedTasks := false
	hasCheckboxTasks := false

	for _, task := range allTasks {
		if strings.Contains(task.Title, "Configure") || strings.Contains(task.Title, "Install") || strings.Contains(task.Title, "Setup database") {
			hasNumberedTasks = true
		}
		if strings.Contains(task.Title, "Implement") || strings.Contains(task.Title, "Setup completed") {
			hasCheckboxTasks = true
		}
	}

	if !hasNumberedTasks {
		t.Error("Expected to extract numbered tasks")
	}

	if !hasCheckboxTasks {
		t.Error("Expected to extract checkbox tasks")
	}
}

func TestParseComplexDocument(t *testing.T) {
	// Test with the real parallel-coordination-design.md document
	testFile := "../../harness/docs/research/parallel-coordination-design.md"

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Skipping test: parallel-coordination-design.md not found")
	}

	doc, err := ParsePlanDocument(testFile)
	if err != nil {
		t.Fatalf("Failed to parse complex document: %v", err)
	}

	t.Logf("Parsed complex document: %s", doc.Title)
	t.Logf("Metadata: Version=%s, Status=%s, Date=%s, Author=%s",
		doc.Metadata.Version, doc.Metadata.Status, doc.Metadata.Date, doc.Metadata.Author)
	t.Logf("Sections: %d", len(doc.Sections))

	// Extract all tasks
	var allTasks []Task
	var extractFromSection func(s *Section)
	extractFromSection = func(s *Section) {
		tasks := ExtractTasks(s, s.Title)
		allTasks = append(allTasks, tasks...)
		for i := range s.Subsections {
			extractFromSection(&s.Subsections[i])
		}
	}

	for i := range doc.Sections {
		extractFromSection(&doc.Sections[i])
	}

	t.Logf("Total tasks extracted: %d", len(allTasks))

	if len(allTasks) == 0 {
		t.Error("Expected to extract tasks from complex document, got 0")
	}

	// Log some sample tasks
	for i := 0; i < min(5, len(allTasks)); i++ {
		t.Logf("  Sample task %d: %s", i+1, allTasks[i].Title)
	}
}

func TestIntegration_CheckboxTaskExtraction(t *testing.T) {
	// Integration test with the checkbox test file
	testFile := "../../testdata/plans/checkbox-test.md"

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Skipping test: checkbox-test.md not found")
	}

	doc, err := ParsePlanDocument(testFile)
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Convert to epic
	opts := ConversionOptions{
		Prefix:   "test",
		Priority: 2,
	}

	epic, err := ConvertToEpic(doc, opts)
	if err != nil {
		t.Fatalf("Failed to convert to epic: %v", err)
	}

	// Verify we extracted tasks
	if len(epic.Subtasks) == 0 {
		t.Error("Expected to extract tasks from checkbox document")
	}

	t.Logf("Extracted %d tasks from checkbox test document", len(epic.Subtasks))

	// Verify we got tasks from different formats
	hasStandaloneCheckbox := false
	hasTaskSectionCheckbox := false
	hasNumberedTask := false

	for _, task := range epic.Subtasks {
		if strings.Contains(task.Title, "Install prerequisites") ||
		   strings.Contains(task.Title, "Configure environment") {
			hasStandaloneCheckbox = true
		}
		if strings.Contains(task.Title, "Implement feature") {
			hasTaskSectionCheckbox = true
		}
		if strings.Contains(task.Title, "Review code") ||
		   strings.Contains(task.Title, "Fix bugs") {
			hasNumberedTask = true
		}
	}

	if !hasStandaloneCheckbox {
		t.Error("Expected to extract standalone checkbox tasks")
	}

	if !hasTaskSectionCheckbox {
		t.Error("Expected to extract tasks from **Tasks:** sections")
	}

	if !hasNumberedTask {
		t.Error("Expected to extract numbered tasks")
	}

	t.Log("✓ Successfully extracted tasks from all formats:")
	t.Log("  - Standalone checkboxes")
	t.Log("  - **Tasks:** section checkboxes")
	t.Log("  - Numbered task lists")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

		// Debug: show what sections were parsed
		t.Logf("Debug: Document has %d sections", len(doc.Sections))
		for i, section := range doc.Sections {
			t.Logf("  Section %d: %s (type: %s, level: %d)", i+1, section.Title, section.Type, section.Level)
			tasks := ExtractTasks(&section, section.Title)
			t.Logf("    Tasks extracted: %d", len(tasks))
			for j, task := range tasks {
				t.Logf("      Task %d: %s", j+1, task.Title)
			}
		}
	}

	t.Logf("Epic created: %s", epic.Title)
	t.Logf("Metadata: Version=%s, Status=%s", epic.Description, epic.Status)
	t.Logf("Subtasks: %d", len(epic.Subtasks))
	for i, task := range epic.Subtasks {
		t.Logf("  Task %d: [%s] %s (phase: %s)", i+1, task.ID, task.Title, task.Phase)
		if len(task.Description) > 100 {
			t.Logf("    Description: %s...", task.Description[:100])
		} else {
			t.Logf("    Description: %s", task.Description)
		}
	}
}
