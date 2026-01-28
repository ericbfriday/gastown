// Package sources provides data source adapters for plan-oracle.
package sources

import (
	"math"
	"sort"

	"github.com/steveyegge/gastown/internal/planoracle/models"
)

// MetricsCollector collects and aggregates historical metrics.
type MetricsCollector struct {
	beadsSource *BeadsSource
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector(beadsSource *BeadsSource) *MetricsCollector {
	return &MetricsCollector{
		beadsSource: beadsSource,
	}
}

// CollectMetrics collects metrics from all completed work items.
func (mc *MetricsCollector) CollectMetrics() (*models.HistoricalMetrics, error) {
	metrics := models.NewHistoricalMetrics()

	// Load all completed work items
	completedItems, err := mc.beadsSource.LoadCompletedWorkItems()
	if err != nil {
		return nil, err
	}

	// Group by type for aggregate calculations
	byType := make(map[string][]*models.WorkItem)
	byRig := make(map[string][]*models.WorkItem)

	for _, item := range completedItems {
		// Only include items with actual completion data
		if !item.IsComplete() || item.ActualDays == 0 {
			continue
		}

		// Store item metrics
		itemMetric := &models.ItemMetrics{
			ID:               item.ID,
			Type:             item.Type,
			ActualDays:       item.ActualDays,
			EstimatedDays:    item.EstimatedDays,
			EstimateAccuracy: item.EstimateAccuracy(),
			ComplexityScore:  item.Complexity,
			DependencyCount:  len(item.Dependencies),
			RigAffinity:      item.RigAffinity,
		}
		metrics.AddItemMetric(itemMetric)

		// Group by type
		byType[item.Type] = append(byType[item.Type], item)

		// Group by rig
		for _, rig := range item.RigAffinity {
			byRig[rig] = append(byRig[rig], item)
		}
	}

	// Calculate type averages
	for issueType, items := range byType {
		metrics.TypeAverages[issueType] = calculateTypeMetrics(issueType, items)
	}

	// Calculate rig velocity
	for rig, items := range byRig {
		metrics.RigVelocity[rig] = calculateRigVelocity(items)
	}

	return metrics, nil
}

// calculateTypeMetrics calculates aggregate metrics for a work item type.
func calculateTypeMetrics(issueType string, items []*models.WorkItem) *models.TypeMetrics {
	if len(items) == 0 {
		return &models.TypeMetrics{
			Type:       issueType,
			SampleSize: 0,
		}
	}

	// Extract actual days
	days := make([]float64, 0, len(items))
	for _, item := range items {
		if item.ActualDays > 0 {
			days = append(days, item.ActualDays)
		}
	}

	if len(days) == 0 {
		return &models.TypeMetrics{
			Type:       issueType,
			SampleSize: 0,
		}
	}

	// Sort for percentile calculations
	sort.Float64s(days)

	// Calculate statistics
	avg := average(days)
	median := percentile(days, 0.5)
	stdDev := standardDeviation(days, avg)

	return &models.TypeMetrics{
		Type:        issueType,
		AverageDays: avg,
		MedianDays:  median,
		StdDev:      stdDev,
		SampleSize:  len(days),
		P50:         percentile(days, 0.5),
		P75:         percentile(days, 0.75),
		P90:         percentile(days, 0.9),
	}
}

// calculateRigVelocity calculates average days per work item for a rig.
func calculateRigVelocity(items []*models.WorkItem) float64 {
	if len(items) == 0 {
		return 0
	}

	total := 0.0
	count := 0

	for _, item := range items {
		if item.ActualDays > 0 {
			total += item.ActualDays
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / float64(count)
}

// average calculates the mean of a slice of floats.
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// percentile calculates the p-th percentile of a sorted slice.
// p should be between 0.0 and 1.0.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}

	index := p * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// standardDeviation calculates the standard deviation.
func standardDeviation(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}

	variance /= float64(len(values))
	return math.Sqrt(variance)
}
