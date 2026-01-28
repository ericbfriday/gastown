// Package models defines data models for the plan-oracle plugin.
package models

import (
	"time"

	"github.com/steveyegge/gastown/internal/beads"
)

// WorkItem extends beads.Issue with planning metadata.
type WorkItem struct {
	// Core fields from beads.Issue
	ID          string
	Title       string
	Description string
	Status      string
	Priority    int
	Type        string // task, feature, epic, convoy
	Parent      string
	Children    []string
	CreatedAt   time.Time
	ClosedAt    *time.Time

	// Dependency information
	Dependencies []Dependency
	Dependents   []Dependency
	BlockedBy    []string
	Blocks       []string

	// Planning metadata
	Complexity    ComplexityScore
	EstimatedDays float64
	ActualDays    float64
	RigAffinity   []string // Rigs this work touches

	// Risk assessment
	RiskLevel   RiskLevel
	RiskFactors []RiskFactor

	// Decomposition
	DecomposedFrom string   // Parent if this was decomposed
	Subtasks       []string // Children from decomposition
}

// Dependency represents a relationship between work items.
type Dependency struct {
	IssueID  string
	Type     string  // "blocks", "depends_on", "related"
	Strength float64 // 0.0-1.0, how critical is this dependency
}

// ComplexityScore represents multi-dimensional complexity assessment.
type ComplexityScore struct {
	Size          int     // 1-5 (XS, S, M, L, XL)
	TechnicalRisk int     // 1-5
	CrossRigRisk  int     // 1-5
	TestingBurden int     // 1-5
	Total         float64 // Weighted combination
}

// RiskLevel represents the overall risk of a work item.
type RiskLevel int

const (
	RiskLow RiskLevel = iota
	RiskMedium
	RiskHigh
	RiskCritical
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// RiskFactor represents a specific risk and its mitigation.
type RiskFactor struct {
	Category    string // "technical", "dependency", "coordination", "unknown"
	Description string
	Mitigation  string
}

// FromBeadsIssue converts a beads.Issue to a WorkItem.
func FromBeadsIssue(issue *beads.Issue) *WorkItem {
	w := &WorkItem{
		ID:          issue.ID,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      issue.Status,
		Priority:    issue.Priority,
		Type:        issue.Type,
		Parent:      issue.Parent,
		Children:    issue.Children,
		Dependencies: make([]Dependency, 0),
		Dependents:   make([]Dependency, 0),
		BlockedBy:    issue.BlockedBy,
		Blocks:       issue.Blocks,
		RigAffinity:  extractRigAffinity(issue),
		RiskFactors:  make([]RiskFactor, 0),
		Subtasks:     make([]string, 0),
	}

	// Parse timestamps
	if t, err := time.Parse(time.RFC3339, issue.CreatedAt); err == nil {
		w.CreatedAt = t
	}
	if issue.ClosedAt != "" {
		if t, err := time.Parse(time.RFC3339, issue.ClosedAt); err == nil {
			w.ClosedAt = &t
		}
	}

	// Calculate actual days if closed
	if w.ClosedAt != nil {
		w.ActualDays = w.ClosedAt.Sub(w.CreatedAt).Hours() / 24.0
	}

	// Convert dependencies from beads format
	for _, dep := range issue.Dependencies {
		w.Dependencies = append(w.Dependencies, Dependency{
			IssueID:  dep.ID,
			Type:     dep.DependencyType,
			Strength: 1.0, // Default to critical
		})
	}

	for _, dep := range issue.Dependents {
		w.Dependents = append(w.Dependents, Dependency{
			IssueID:  dep.ID,
			Type:     dep.DependencyType,
			Strength: 1.0,
		})
	}

	return w
}

// extractRigAffinity extracts rig names from issue ID prefix.
func extractRigAffinity(issue *beads.Issue) []string {
	// Issue IDs are prefixed with rig name (e.g., "gt-abc", "bd-xyz")
	// Extract the prefix before the first hyphen
	if len(issue.ID) > 0 {
		for i, c := range issue.ID {
			if c == '-' {
				return []string{issue.ID[:i]}
			}
		}
	}
	return []string{"unknown"}
}

// IsComplete returns true if the work item is closed.
func (w *WorkItem) IsComplete() bool {
	return w.Status == "closed" || w.ClosedAt != nil
}

// IsBlocked returns true if the work item has blocking dependencies.
func (w *WorkItem) IsBlocked() bool {
	return len(w.BlockedBy) > 0
}

// EstimateAccuracy returns the ratio of estimated to actual days (if complete).
func (w *WorkItem) EstimateAccuracy() float64 {
	if !w.IsComplete() || w.EstimatedDays == 0 {
		return 0
	}
	return w.EstimatedDays / w.ActualDays
}
