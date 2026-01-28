// Package models defines data models for the plan-oracle plugin.
package models

// HistoricalMetrics stores aggregated historical data for estimation.
type HistoricalMetrics struct {
	WorkItemMetrics map[string]*ItemMetrics
	TypeAverages    map[string]*TypeMetrics
	RigVelocity     map[string]float64
}

// ItemMetrics stores metrics for a specific work item.
type ItemMetrics struct {
	ID               string
	Type             string
	ActualDays       float64
	EstimatedDays    float64
	EstimateAccuracy float64 // Ratio of estimated to actual
	ComplexityScore  ComplexityScore
	DependencyCount  int
	RigAffinity      []string
}

// TypeMetrics stores aggregate metrics for a work item type.
type TypeMetrics struct {
	Type        string
	AverageDays float64
	MedianDays  float64
	StdDev      float64
	SampleSize  int
	P50         float64 // 50th percentile
	P75         float64 // 75th percentile
	P90         float64 // 90th percentile
}

// NewHistoricalMetrics creates an empty metrics structure.
func NewHistoricalMetrics() *HistoricalMetrics {
	return &HistoricalMetrics{
		WorkItemMetrics: make(map[string]*ItemMetrics),
		TypeAverages:    make(map[string]*TypeMetrics),
		RigVelocity:     make(map[string]float64),
	}
}

// AddItemMetric adds metrics for a completed work item.
func (h *HistoricalMetrics) AddItemMetric(metric *ItemMetrics) {
	h.WorkItemMetrics[metric.ID] = metric
}

// GetTypeMetrics returns aggregate metrics for a given type.
func (h *HistoricalMetrics) GetTypeMetrics(issueType string) *TypeMetrics {
	if tm, ok := h.TypeAverages[issueType]; ok {
		return tm
	}
	// Return defaults if no data
	return &TypeMetrics{
		Type:        issueType,
		AverageDays: 0,
		MedianDays:  0,
		SampleSize:  0,
	}
}

// GetRigVelocity returns the average days per work item for a rig.
func (h *HistoricalMetrics) GetRigVelocity(rig string) float64 {
	if velocity, ok := h.RigVelocity[rig]; ok {
		return velocity
	}
	return 0 // Unknown rig
}

// FindSimilarWork returns work items similar to the given characteristics.
func (h *HistoricalMetrics) FindSimilarWork(issueType string, complexity ComplexityScore, limit int) []*ItemMetrics {
	similar := make([]*ItemMetrics, 0)

	for _, metric := range h.WorkItemMetrics {
		if metric.Type == issueType {
			// Simple similarity: based on complexity total
			diff := abs(metric.ComplexityScore.Total - complexity.Total)
			if diff < 1.0 { // Within 1 point of complexity
				similar = append(similar, metric)
			}
		}
	}

	// Return up to limit items
	if len(similar) > limit {
		return similar[:limit]
	}
	return similar
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
