// Package mergeoracle provides intelligent analysis of merge queue safety.
//
// This package analyzes merge requests to predict conflicts, assess risk,
// evaluate test coverage, and recommend optimal merge timing.
package mergeoracle

import (
	"time"

	"github.com/steveyegge/gastown/internal/refinery"
)

// RiskLevel categorizes the overall risk of a merge.
type RiskLevel string

const (
	// RiskLow indicates safe to merge anytime (0-30).
	RiskLow RiskLevel = "low"

	// RiskMedium indicates merge with caution (31-50).
	RiskMedium RiskLevel = "medium"

	// RiskHigh indicates address concerns before merging (51-70).
	RiskHigh RiskLevel = "high"

	// RiskCritical indicates do not merge (71-100).
	RiskCritical RiskLevel = "critical"
)

// MRAnalysis contains comprehensive analysis of a merge request.
type MRAnalysis struct {
	// MR is the merge request being analyzed.
	MR *refinery.MRInfo

	// RiskScore is the overall risk score (0-100, lower is safer).
	RiskScore int

	// RiskLevel is the categorized risk level.
	RiskLevel RiskLevel

	// ConflictRisk is the risk of merge conflicts (0-30).
	ConflictRisk ConflictRisk

	// TestRisk is the risk related to testing (0-20).
	TestRisk TestRisk

	// SizeRisk is the risk related to change size (0-15).
	SizeRisk SizeRisk

	// DependencyRisk is the risk related to dependencies (0-10).
	DependencyRisk DependencyRisk

	// HistoryRisk is the risk based on historical patterns (0-5).
	HistoryRisk HistoryRisk

	// Conflicts lists predicted conflicts with other MRs.
	Conflicts []ConflictPrediction

	// Recommendations are actionable suggestions to reduce risk.
	Recommendations []Recommendation

	// OptimalWindow suggests the best time to merge.
	OptimalWindow *MergeWindow

	// AnalyzedAt is when this analysis was performed.
	AnalyzedAt time.Time
}

// ConflictRisk represents conflict-related risk factors.
type ConflictRisk struct {
	// Score is the conflict risk score (0-30).
	Score int

	// OverlappingMRs is the number of pending MRs with file overlap.
	OverlappingMRs int

	// RecentConflicts is the number of recent conflicts in changed files.
	RecentConflicts int

	// TargetDivergence is commits ahead on target branch.
	TargetDivergence int

	// MergeBaseAge is days since merge base.
	MergeBaseAge int

	// Details provides specific conflict information.
	Details string
}

// TestRisk represents testing-related risk factors.
type TestRisk struct {
	// Score is the test risk score (0-20).
	Score int

	// HasTests indicates if test changes are included.
	HasTests bool

	// FailingTests indicates if tests are currently failing.
	FailingTests bool

	// CoverageDelta is the estimated coverage change.
	CoverageDelta float64

	// FlakyHistory indicates if touched files have flaky test history.
	FlakyHistory bool

	// Details provides specific test information.
	Details string
}

// SizeRisk represents size-related risk factors.
type SizeRisk struct {
	// Score is the size risk score (0-15).
	Score int

	// LinesChanged is the total lines added/removed.
	LinesChanged int

	// FilesChanged is the number of files modified.
	FilesChanged int

	// SubsystemsAffected is the number of subsystems touched.
	SubsystemsAffected int

	// Details provides specific size information.
	Details string
}

// DependencyRisk represents dependency-related risk factors.
type DependencyRisk struct {
	// Score is the dependency risk score (0-10).
	Score int

	// BlocksOthers is the number of MRs blocked by this one.
	BlocksOthers int

	// BlockedBy indicates if this MR is blocked by failing MRs.
	BlockedBy []string

	// ConvoyFragmented indicates if convoy is partially merged.
	ConvoyFragmented bool

	// Details provides specific dependency information.
	Details string
}

// HistoryRisk represents historical pattern-based risk factors.
type HistoryRisk struct {
	// Score is the history risk score (0-5).
	Score int

	// AuthorFailureRate is the percentage of recent failed merges.
	AuthorFailureRate float64

	// FileVelocity is the change frequency of modified files.
	FileVelocity float64

	// Details provides specific history information.
	Details string
}

// ConflictPrediction represents a predicted merge conflict.
type ConflictPrediction struct {
	// WithMR is the ID of the conflicting merge request.
	WithMR string

	// WithBranch is the branch name of the conflict.
	WithBranch string

	// Files are the files predicted to conflict.
	Files []string

	// Severity indicates how serious the conflict is.
	Severity ConflictSeverity

	// Confidence is how confident the prediction is (0.0-1.0).
	Confidence float64

	// Type is the kind of conflict predicted.
	Type ConflictType

	// Suggestions are recommendations to avoid the conflict.
	Suggestions []string
}

// ConflictSeverity indicates how serious a conflict is.
type ConflictSeverity string

const (
	// SeverityLow indicates minor conflict, easy to resolve.
	SeverityLow ConflictSeverity = "low"

	// SeverityMedium indicates moderate conflict, requires review.
	SeverityMedium ConflictSeverity = "medium"

	// SeverityHigh indicates serious conflict, significant merge work.
	SeverityHigh ConflictSeverity = "high"
)

// ConflictType categorizes the kind of conflict.
type ConflictType string

const (
	// ConflictDirect means same files modified.
	ConflictDirect ConflictType = "direct"

	// ConflictSemantic means related code modified.
	ConflictSemantic ConflictType = "semantic"

	// ConflictBuild means package/dependency conflicts.
	ConflictBuild ConflictType = "build"

	// ConflictTest means test-only conflicts (lower risk).
	ConflictTest ConflictType = "test"
)

// Recommendation provides actionable advice.
type Recommendation struct {
	// Priority indicates importance (1=highest).
	Priority int

	// Category groups recommendations.
	Category RecommendationCategory

	// Message is the recommendation text.
	Message string

	// Action is the suggested action (if any).
	Action string
}

// RecommendationCategory groups recommendations.
type RecommendationCategory string

const (
	// CategoryConflict relates to conflict avoidance.
	CategoryConflict RecommendationCategory = "conflict"

	// CategoryTesting relates to test coverage.
	CategoryTesting RecommendationCategory = "testing"

	// CategorySize relates to change size.
	CategorySize RecommendationCategory = "size"

	// CategoryTiming relates to merge timing.
	CategoryTiming RecommendationCategory = "timing"

	// CategoryDependency relates to dependencies.
	CategoryDependency RecommendationCategory = "dependency"
)

// MergeWindow suggests an optimal time to merge.
type MergeWindow struct {
	// After indicates MRs that should merge first.
	After []string

	// Before indicates MRs that should merge after.
	Before []string

	// EstimatedWait is the estimated wait time.
	EstimatedWait time.Duration

	// Reasoning explains why this window is optimal.
	Reasoning string
}

// QueueAnalysis contains analysis of the entire merge queue.
type QueueAnalysis struct {
	// MRs is the list of analyzed merge requests.
	MRs []*MRAnalysis

	// TotalRisk is the aggregate queue risk.
	TotalRisk int

	// ConflictPairs lists all predicted conflicts.
	ConflictPairs []ConflictPair

	// RecommendedOrder suggests optimal merge order.
	RecommendedOrder []string

	// QueueHealth is an overall queue health assessment.
	QueueHealth QueueHealth

	// AnalyzedAt is when this analysis was performed.
	AnalyzedAt time.Time
}

// ConflictPair represents a pair of MRs that may conflict.
type ConflictPair struct {
	// MR1 is the first merge request ID.
	MR1 string

	// MR2 is the second merge request ID.
	MR2 string

	// Files are the conflicting files.
	Files []string

	// Severity indicates conflict severity.
	Severity ConflictSeverity
}

// QueueHealth represents overall queue health metrics.
type QueueHealth struct {
	// Status is the overall queue status.
	Status QueueStatus

	// HighRiskCount is the number of high/critical risk MRs.
	HighRiskCount int

	// ConflictCount is the number of potential conflicts.
	ConflictCount int

	// BlockedCount is the number of blocked MRs.
	BlockedCount int

	// AverageAge is the average MR age.
	AverageAge time.Duration

	// Recommendations are queue-level recommendations.
	Recommendations []string
}

// QueueStatus represents overall queue health.
type QueueStatus string

const (
	// QueueHealthy indicates no significant issues.
	QueueHealthy QueueStatus = "healthy"

	// QueueWarning indicates some concerns.
	QueueWarning QueueStatus = "warning"

	// QueueCritical indicates serious issues.
	QueueCritical QueueStatus = "critical"
)

// AnalysisConfig configures analysis behavior.
type AnalysisConfig struct {
	// IncludeHistory enables historical pattern analysis.
	IncludeHistory bool

	// IncludeTesting enables test coverage analysis.
	IncludeTesting bool

	// IncludeConflicts enables conflict prediction.
	IncludeConflicts bool

	// MaxHistoryDays limits historical analysis depth.
	MaxHistoryDays int

	// ConflictThreshold is minimum confidence for conflict warnings.
	ConflictThreshold float64
}

// DefaultAnalysisConfig returns sensible defaults.
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		IncludeHistory:    true,
		IncludeTesting:    true,
		IncludeConflicts:  true,
		MaxHistoryDays:    30,
		ConflictThreshold: 0.5,
	}
}

// GetRiskLevel returns the risk level for a given score.
func GetRiskLevel(score int) RiskLevel {
	switch {
	case score <= 30:
		return RiskLow
	case score <= 50:
		return RiskMedium
	case score <= 70:
		return RiskHigh
	default:
		return RiskCritical
	}
}

// RiskIcon returns an emoji icon for the risk level.
func (r RiskLevel) Icon() string {
	switch r {
	case RiskLow:
		return "🟢"
	case RiskMedium:
		return "🟡"
	case RiskHigh:
		return "🟠"
	case RiskCritical:
		return "🔴"
	default:
		return "⚪"
	}
}

// String returns a human-readable risk level.
func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "Low Risk"
	case RiskMedium:
		return "Medium Risk"
	case RiskHigh:
		return "High Risk"
	case RiskCritical:
		return "Critical Risk"
	default:
		return "Unknown"
	}
}
