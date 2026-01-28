// Package analyzer provides analysis engines for the plan-oracle plugin.
package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/steveyegge/gastown/internal/planoracle/models"
)

// Decomposer breaks down large work items into smaller subtasks.
type Decomposer struct {
	metrics models.HistoricalMetrics
}

// NewDecomposer creates a new decomposer with historical metrics.
func NewDecomposer(metrics *models.HistoricalMetrics) *Decomposer {
	return &Decomposer{
		metrics: *metrics,
	}
}

// Decompose breaks down a work item into subtasks.
func (d *Decomposer) Decompose(item *models.WorkItem) (*models.DecompositionResult, error) {
	if item.Type != "epic" && item.Type != "feature" && item.Type != "convoy" {
		return nil, fmt.Errorf("can only decompose epics, features, or convoys (got: %s)", item.Type)
	}

	// Try different decomposition strategies
	result := &models.DecompositionResult{
		ParentID: item.ID,
		Subtasks: make([]models.SubTask, 0),
	}

	// Strategy 1: Extract from markdown task list
	if tasks := d.extractMarkdownTasks(item.Description); len(tasks) > 0 {
		result.Subtasks = tasks
		result.Source = "markdown"
		result.Confidence = 0.9
	}

	// Strategy 2: Extract from steps/phases section
	if len(result.Subtasks) == 0 {
		if tasks := d.extractSteps(item.Description); len(tasks) > 0 {
			result.Subtasks = tasks
			result.Source = "steps"
			result.Confidence = 0.8
		}
	}

	// Strategy 3: Use type-based template
	if len(result.Subtasks) == 0 {
		if tasks := d.applyTemplate(item); len(tasks) > 0 {
			result.Subtasks = tasks
			result.Source = "template"
			result.Confidence = 0.6
		}
	}

	// Strategy 4: Analyze historical patterns
	if len(result.Subtasks) == 0 {
		if tasks := d.applyHistoricalPattern(item); len(tasks) > 0 {
			result.Subtasks = tasks
			result.Source = "historical"
			result.Confidence = 0.7
		}
	}

	// Estimate each subtask
	for i := range result.Subtasks {
		result.Subtasks[i].EstimatedDays = d.estimateSubtask(&result.Subtasks[i], item)
		result.TotalEstimate += result.Subtasks[i].EstimatedDays
	}

	return result, nil
}

// extractMarkdownTasks extracts tasks from markdown checkbox lists.
func (d *Decomposer) extractMarkdownTasks(description string) []models.SubTask {
	tasks := make([]models.SubTask, 0)

	// Match markdown task list items: - [ ] or - [x] or * [ ]
	taskPattern := regexp.MustCompile(`(?m)^[\s]*[-*]\s*\[([ xX])\]\s*(.+)$`)
	matches := taskPattern.FindAllStringSubmatch(description, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			title := strings.TrimSpace(match[2])
			if title != "" {
				tasks = append(tasks, models.SubTask{
					Title:    title,
					Type:     "task",
					Priority: 2, // Default medium priority
				})
			}
		}
	}

	return tasks
}

// extractSteps extracts tasks from numbered steps or phases sections.
func (d *Decomposer) extractSteps(description string) []models.SubTask {
	tasks := make([]models.SubTask, 0)

	// Look for sections like "## Steps" or "## Phases" or "## Tasks"
	sectionPattern := regexp.MustCompile(`(?mi)^##\s+(steps?|phases?|tasks?)[\s]*$`)
	stepPattern := regexp.MustCompile(`(?m)^\s*\d+\.\s+(.+)$`)

	// Find section header
	sectionMatch := sectionPattern.FindStringIndex(description)
	if sectionMatch == nil {
		return tasks
	}

	// Extract content after section header
	content := description[sectionMatch[1]:]

	// Find next section (stop there)
	nextSection := regexp.MustCompile(`(?m)^##\s+`).FindStringIndex(content)
	if nextSection != nil {
		content = content[:nextSection[0]]
	}

	// Extract numbered steps
	matches := stepPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			title := strings.TrimSpace(match[1])
			if title != "" {
				tasks = append(tasks, models.SubTask{
					Title:    title,
					Type:     "task",
					Priority: 2,
				})
			}
		}
	}

	return tasks
}

// applyTemplate applies a decomposition template based on work item type.
func (d *Decomposer) applyTemplate(item *models.WorkItem) []models.SubTask {
	tasks := make([]models.SubTask, 0)

	switch item.Type {
	case "epic":
		// Generic epic template
		tasks = []models.SubTask{
			{Title: "Design and architecture", Type: "task", Priority: 2},
			{Title: "Core implementation", Type: "task", Priority: 2},
			{Title: "Testing and validation", Type: "task", Priority: 2},
			{Title: "Documentation", Type: "task", Priority: 3},
		}

	case "feature":
		// Generic feature template
		tasks = []models.SubTask{
			{Title: "Implementation", Type: "task", Priority: 2},
			{Title: "Unit tests", Type: "task", Priority: 2},
			{Title: "Integration testing", Type: "task", Priority: 3},
		}

	case "convoy":
		// Convoy: Extract legs from description if possible
		// Otherwise use generic parallel work template
		tasks = []models.SubTask{
			{Title: "Leg 1: (parallel work)", Type: "task", Priority: 2},
			{Title: "Leg 2: (parallel work)", Type: "task", Priority: 2},
			{Title: "Synthesis", Type: "task", Priority: 2},
		}
	}

	return tasks
}

// applyHistoricalPattern finds similar completed work and applies its pattern.
func (d *Decomposer) applyHistoricalPattern(item *models.WorkItem) []models.SubTask {
	// Find similar historical work
	similar := d.metrics.FindSimilarWork(item.Type, item.Complexity, 5)

	if len(similar) == 0 {
		return nil
	}

	// For now, return empty - this would require storing decomposition patterns
	// in historical metrics, which is a future enhancement
	return nil
}

// estimateSubtask estimates the effort for a subtask.
func (d *Decomposer) estimateSubtask(subtask *models.SubTask, parent *models.WorkItem) float64 {
	// Base estimate by keyword matching
	title := strings.ToLower(subtask.Title)

	// Simple heuristics based on common task types
	if strings.Contains(title, "design") || strings.Contains(title, "architecture") {
		return 2.0 // 2 days for design
	}
	if strings.Contains(title, "implement") || strings.Contains(title, "core") {
		return 3.0 // 3 days for core implementation
	}
	if strings.Contains(title, "test") {
		return 1.0 // 1 day for testing
	}
	if strings.Contains(title, "document") || strings.Contains(title, "docs") {
		return 0.5 // Half day for documentation
	}
	if strings.Contains(title, "refactor") || strings.Contains(title, "cleanup") {
		return 1.5 // 1.5 days for refactoring
	}

	// Default: small task
	return 1.0
}
