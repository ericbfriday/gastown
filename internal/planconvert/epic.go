package planconvert

import (
	"fmt"
	"strings"
	"time"
)

// ConvertToEpic converts a parsed plan document into a beads epic structure.
func ConvertToEpic(doc *PlanDocument, opts ConversionOptions) (*Epic, error) {
	if opts.Priority == 0 {
		opts.Priority = 2 // Default priority
	}

	now := time.Now()

	epic := &Epic{
		ID:           generateID(opts.Prefix, "epic", 0),
		Title:        doc.Title,
		Description:  buildEpicDescription(doc, opts),
		Status:       "open",
		Priority:     opts.Priority,
		IssueType:    "epic",
		CreatedAt:    now,
		UpdatedAt:    now,
		SourceFile:   doc.FilePath,
		Subtasks:     []Bead{},
		Dependencies: []Dependency{},
	}

	// Process all sections to extract tasks
	taskCounter := 0
	for _, section := range doc.Sections {
		tasks := processSection(&section, doc.Title, opts.Prefix, &taskCounter, opts)
		epic.Subtasks = append(epic.Subtasks, tasks...)
	}

	// Link subtasks to epic via dependencies
	for i := range epic.Subtasks {
		epic.Subtasks[i].Dependencies = append(epic.Subtasks[i].Dependencies, Dependency{
			IssueID:     epic.Subtasks[i].ID,
			DependsOnID: epic.ID,
			Type:        "blocks",
			CreatedAt:   now,
			CreatedBy:   "plan-converter",
		})
	}

	return epic, nil
}

// processSection recursively processes sections to extract tasks.
func processSection(section *Section, epicTitle, prefix string, counter *int, opts ConversionOptions) []Bead {
	var beads []Bead

	// Extract tasks from this section
	tasks := ExtractTasks(section, section.Title)
	for _, task := range tasks {
		*counter++
		bead := Bead{
			ID:          generateID(prefix, "task", *counter),
			Title:       task.Title,
			Description: buildTaskDescription(task, section, epicTitle, opts),
			Status:      "open",
			Priority:    task.Priority,
			IssueType:   "task",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Phase:       task.Phase,
			Order:       task.Order,
		}
		beads = append(beads, bead)
	}

	// Process subsections
	for i := range section.Subsections {
		subBeads := processSection(&section.Subsections[i], epicTitle, prefix, counter, opts)
		beads = append(beads, subBeads...)
	}

	return beads
}

// buildEpicDescription builds the epic description with context.
func buildEpicDescription(doc *PlanDocument, opts ConversionOptions) string {
	var desc strings.Builder

	desc.WriteString(fmt.Sprintf("Auto-generated from: %s\n\n", doc.FilePath))

	if doc.Metadata.Version != "" {
		desc.WriteString(fmt.Sprintf("**Version:** %s\n", doc.Metadata.Version))
	}
	if doc.Metadata.Status != "" {
		desc.WriteString(fmt.Sprintf("**Status:** %s\n", doc.Metadata.Status))
	}
	if doc.Metadata.Date != "" {
		desc.WriteString(fmt.Sprintf("**Date:** %s\n", doc.Metadata.Date))
	}
	if doc.Metadata.Author != "" {
		desc.WriteString(fmt.Sprintf("**Author:** %s\n", doc.Metadata.Author))
	}

	desc.WriteString("\n## Overview\n\n")
	desc.WriteString(fmt.Sprintf("This epic tracks implementation of: %s\n\n", doc.Title))

	// Add phase summary
	phaseCount := 0
	for _, section := range doc.Sections {
		if section.Type == SectionTypePhase {
			phaseCount++
		}
	}

	if phaseCount > 0 {
		desc.WriteString(fmt.Sprintf("**Phases:** %d\n", phaseCount))
	}

	desc.WriteString("\n## Source Document\n\n")
	desc.WriteString(fmt.Sprintf("See full design: %s\n", doc.FilePath))

	return desc.String()
}

// buildTaskDescription builds task description with context.
func buildTaskDescription(task Task, section *Section, epicTitle string, opts ConversionOptions) string {
	var desc strings.Builder

	// Add context
	desc.WriteString(fmt.Sprintf("**Epic:** %s\n", epicTitle))
	if task.Phase != "" {
		desc.WriteString(fmt.Sprintf("**Phase:** %s\n", task.Phase))
	}

	desc.WriteString("\n## Task\n\n")
	desc.WriteString(task.Title)
	desc.WriteString("\n")

	// Add deliverables if present
	if len(task.Deliverables) > 0 {
		desc.WriteString("\n## Deliverables\n\n")
		for _, d := range task.Deliverables {
			desc.WriteString(fmt.Sprintf("- [ ] %s\n", d))
		}
	}

	// Add success criteria if present
	if len(task.Criteria) > 0 {
		desc.WriteString("\n## Success Criteria\n\n")
		for _, c := range task.Criteria {
			desc.WriteString(fmt.Sprintf("- %s\n", c))
		}
	}

	// Add dependencies if present
	if len(task.Dependencies) > 0 {
		desc.WriteString("\n## Dependencies\n\n")
		for _, dep := range task.Dependencies {
			desc.WriteString(fmt.Sprintf("- %s\n", dep))
		}
	}

	return desc.String()
}

// generateID generates a beads-compatible ID.
func generateID(prefix, idType string, counter int) string {
	if prefix == "" {
		prefix = "plan"
	}

	// Generate a simple sequential ID
	if counter == 0 {
		return fmt.Sprintf("%s-%s", prefix, generateShortID())
	}

	return fmt.Sprintf("%s-%s-%d", prefix, generateShortID(), counter)
}

// generateShortID generates a short random ID similar to beads format.
func generateShortID() string {
	// Simple base36 encoding of timestamp for uniqueness
	// In production, this would use the same ID generation as beads
	now := time.Now().UnixNano()
	const charset = "0123456789abcdefghijklmnopqrstuvwxyz"

	var result strings.Builder
	for i := 0; i < 5; i++ {
		result.WriteByte(charset[now%36])
		now /= 36
	}

	return result.String()
}
