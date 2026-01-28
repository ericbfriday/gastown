// Package models defines data models for the plan-oracle plugin.
package models

// ExecutionPlan represents a recommended execution order for work items.
type ExecutionPlan struct {
	WorkItemID     string
	Phases         []Phase
	CriticalPath   []string // Sequence of work item IDs
	Parallelizable []ParallelSet
	TotalEstimate  float64 // Total days
	RiskProfile    RiskProfile
}

// Phase represents a logical grouping of work items that should be done together.
type Phase struct {
	ID           string
	Name         string
	Items        []string // Work item IDs
	Dependencies []string // Phase IDs this phase depends on
	Estimate     float64  // Days
}

// ParallelSet represents work items that can be done in parallel.
type ParallelSet struct {
	Items      []string // Work items that can be done in parallel
	Constraint string   // Why they're grouped (e.g., "same rig")
}

// RiskProfile represents the overall risk assessment for a plan.
type RiskProfile struct {
	OverallRisk    RiskLevel
	HighRiskItems  []string // Work item IDs
	Mitigations    []RiskFactor
	ContingencyPct float64 // % to add to estimate
}

// SubTask represents a decomposed task from a larger work item.
type SubTask struct {
	Title         string
	Description   string
	EstimatedDays float64
	Priority      int
	Type          string
	Dependencies  []string // Other subtask titles it depends on
}

// DecompositionResult represents the result of decomposing a work item.
type DecompositionResult struct {
	ParentID      string
	Subtasks      []SubTask
	TotalEstimate float64
	Confidence    float64 // 0.0-1.0
	Source        string  // "markdown", "template", "historical", "manual"
}

// EstimateResult represents effort estimation for a work item.
type EstimateResult struct {
	WorkItemID string
	Low        float64 // Low estimate (days)
	High       float64 // High estimate (days)
	Median     float64 // Recommended estimate (days)
	Confidence float64 // 0.0-1.0

	// Breakdown
	Breakdown map[string]float64 // e.g., {"design": 2, "implementation": 10, ...}

	// Supporting data
	HistoricalComparisons []HistoricalComparison
	AdjustmentFactors     []AdjustmentFactor
}

// HistoricalComparison represents a similar historical work item.
type HistoricalComparison struct {
	IssueID       string
	Title         string
	Type          string
	ActualDays    float64
	Similarity    float64 // 0.0-1.0
	SimilarReason string  // Why it's similar
}

// AdjustmentFactor represents a factor that adjusts the base estimate.
type AdjustmentFactor struct {
	Factor      string  // e.g., "new domain", "high complexity"
	Adjustment  float64 // Days to add/subtract
	Reason      string  // Explanation
}

// DependencyGraph represents the full dependency structure.
type DependencyGraph struct {
	Nodes map[string]*GraphNode
	Edges []*GraphEdge
}

// GraphNode represents a work item in the dependency graph.
type GraphNode struct {
	WorkItem     *WorkItem
	Depth        int     // Depth in dependency tree
	CriticalPath bool    // Is this on the critical path?
	EarliestStart float64 // Earliest start time (days from project start)
	LatestStart   float64 // Latest start time without delaying project
}

// GraphEdge represents a dependency between work items.
type GraphEdge struct {
	From     string  // Work item ID
	To       string  // Work item ID
	Type     string  // "blocks", "depends_on", "related"
	Strength float64 // 0.0-1.0
}

// NewDependencyGraph creates an empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*GraphNode),
		Edges: make([]*GraphEdge, 0),
	}
}

// AddNode adds a work item to the graph.
func (g *DependencyGraph) AddNode(item *WorkItem) {
	g.Nodes[item.ID] = &GraphNode{
		WorkItem: item,
		Depth:    -1,
		CriticalPath: false,
	}
}

// AddEdge adds a dependency edge to the graph.
func (g *DependencyGraph) AddEdge(from, to, depType string, strength float64) {
	g.Edges = append(g.Edges, &GraphEdge{
		From:     from,
		To:       to,
		Type:     depType,
		Strength: strength,
	})
}

// GetDependencies returns all work items that the given item depends on.
func (g *DependencyGraph) GetDependencies(itemID string) []*WorkItem {
	deps := make([]*WorkItem, 0)
	for _, edge := range g.Edges {
		if edge.From == itemID {
			if node, ok := g.Nodes[edge.To]; ok {
				deps = append(deps, node.WorkItem)
			}
		}
	}
	return deps
}

// GetDependents returns all work items that depend on the given item.
func (g *DependencyGraph) GetDependents(itemID string) []*WorkItem {
	dependents := make([]*WorkItem, 0)
	for _, edge := range g.Edges {
		if edge.To == itemID {
			if node, ok := g.Nodes[edge.From]; ok {
				dependents = append(dependents, node.WorkItem)
			}
		}
	}
	return dependents
}
