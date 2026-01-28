// Package sources provides data source adapters for plan-oracle.
package sources

import (
	"fmt"

	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/planoracle/models"
)

// BeadsSource provides access to the beads database.
type BeadsSource struct {
	workDir string
}

// NewBeadsSource creates a new beads data source.
func NewBeadsSource(workDir string) *BeadsSource {
	return &BeadsSource{
		workDir: workDir,
	}
}

// LoadWorkItem loads a single work item by ID.
func (b *BeadsSource) LoadWorkItem(id string) (*models.WorkItem, error) {
	issue, err := beads.Show(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load issue %s: %w", id, err)
	}

	return models.FromBeadsIssue(issue), nil
}

// LoadWorkItems loads multiple work items matching the filter.
func (b *BeadsSource) LoadWorkItems(filter beads.ListOptions) ([]*models.WorkItem, error) {
	issues, err := beads.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	workItems := make([]*models.WorkItem, 0, len(issues))
	for i := range issues {
		workItems = append(workItems, models.FromBeadsIssue(&issues[i]))
	}

	return workItems, nil
}

// LoadEpicWithChildren loads an epic and all its child work items.
func (b *BeadsSource) LoadEpicWithChildren(epicID string) ([]*models.WorkItem, error) {
	// Load the epic itself
	epic, err := b.LoadWorkItem(epicID)
	if err != nil {
		return nil, err
	}

	if epic.Type != "epic" && epic.Type != "convoy" && epic.Type != "feature" {
		return nil, fmt.Errorf("%s is not an epic, convoy, or feature (type: %s)", epicID, epic.Type)
	}

	// Load all children
	items := []*models.WorkItem{epic}

	if len(epic.Children) > 0 {
		for _, childID := range epic.Children {
			child, err := b.LoadWorkItem(childID)
			if err != nil {
				// Log but continue - some children may be deleted
				continue
			}
			items = append(items, child)
		}
	}

	return items, nil
}

// LoadCompletedWorkItems loads all completed work items for historical analysis.
func (b *BeadsSource) LoadCompletedWorkItems() ([]*models.WorkItem, error) {
	filter := beads.ListOptions{
		Status: "closed",
	}

	return b.LoadWorkItems(filter)
}

// LoadByType loads all work items of a specific type.
func (b *BeadsSource) LoadByType(issueType string) ([]*models.WorkItem, error) {
	filter := beads.ListOptions{
		Type:   issueType,
		Status: "all",
	}

	return b.LoadWorkItems(filter)
}

// BuildDependencyGraph builds a full dependency graph for the given work items.
func (b *BeadsSource) BuildDependencyGraph(items []*models.WorkItem) *models.DependencyGraph {
	graph := models.NewDependencyGraph()

	// Add all items as nodes
	for _, item := range items {
		graph.AddNode(item)
	}

	// Add edges for dependencies
	for _, item := range items {
		// Add dependency edges
		for _, dep := range item.Dependencies {
			graph.AddEdge(item.ID, dep.IssueID, dep.Type, dep.Strength)
		}

		// Add blocking edges
		for _, blockID := range item.Blocks {
			graph.AddEdge(item.ID, blockID, "blocks", 1.0)
		}
	}

	return graph
}
