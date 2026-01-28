package mergeoracle

import (
	"fmt"
	"sort"
	"time"

	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/refinery"
)

// Analyzer performs merge request analysis.
type Analyzer struct {
	git    *git.Git
	config *AnalysisConfig
}

// NewAnalyzer creates a new merge request analyzer.
func NewAnalyzer(repoPath string, config *AnalysisConfig) (*Analyzer, error) {
	if config == nil {
		config = DefaultAnalysisConfig()
	}

	g := git.NewGit(repoPath)
	if !g.IsRepo() {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}

	return &Analyzer{
		git:    g,
		config: config,
	}, nil
}

// AnalyzeMR performs comprehensive analysis of a single merge request.
func (a *Analyzer) AnalyzeMR(mr *refinery.MRInfo, queue []*refinery.MRInfo) (*MRAnalysis, error) {
	analysis := &MRAnalysis{
		MR:         mr,
		AnalyzedAt: time.Now(),
	}

	// Analyze conflict risk
	conflictRisk, conflicts, err := a.analyzeConflictRisk(mr, queue)
	if err != nil {
		return nil, fmt.Errorf("analyzing conflict risk: %w", err)
	}
	analysis.ConflictRisk = conflictRisk
	analysis.Conflicts = conflicts

	// Analyze test risk
	testRisk, err := a.analyzeTestRisk(mr)
	if err != nil {
		return nil, fmt.Errorf("analyzing test risk: %w", err)
	}
	analysis.TestRisk = testRisk

	// Analyze size risk
	sizeRisk, err := a.analyzeSizeRisk(mr)
	if err != nil {
		return nil, fmt.Errorf("analyzing size risk: %w", err)
	}
	analysis.SizeRisk = sizeRisk

	// Analyze dependency risk
	dependencyRisk, err := a.analyzeDependencyRisk(mr, queue)
	if err != nil {
		return nil, fmt.Errorf("analyzing dependency risk: %w", err)
	}
	analysis.DependencyRisk = dependencyRisk

	// Analyze history risk (if enabled)
	historyRisk := HistoryRisk{Score: 0}
	if a.config.IncludeHistory {
		var err error
		historyRisk, err = a.analyzeHistoryRisk(mr)
		if err != nil {
			// Non-fatal: log warning and continue
			historyRisk = HistoryRisk{Score: 0, Details: fmt.Sprintf("Analysis failed: %v", err)}
		}
	}
	analysis.HistoryRisk = historyRisk

	// Calculate total risk score
	analysis.RiskScore = conflictRisk.Score +
		testRisk.Score +
		sizeRisk.Score +
		dependencyRisk.Score +
		historyRisk.Score

	// Add base risk (minimum 20 points)
	analysis.RiskScore += 20

	// Determine risk level
	analysis.RiskLevel = GetRiskLevel(analysis.RiskScore)

	// Generate recommendations
	analysis.Recommendations = a.generateRecommendations(analysis)

	// Determine optimal merge window
	analysis.OptimalWindow = a.determineOptimalWindow(mr, queue, conflicts)

	return analysis, nil
}

// AnalyzeQueue analyzes the entire merge queue.
func (a *Analyzer) AnalyzeQueue(queue []*refinery.MRInfo) (*QueueAnalysis, error) {
	queueAnalysis := &QueueAnalysis{
		MRs:        make([]*MRAnalysis, 0, len(queue)),
		AnalyzedAt: time.Now(),
	}

	// Analyze each MR
	for _, mr := range queue {
		analysis, err := a.AnalyzeMR(mr, queue)
		if err != nil {
			return nil, fmt.Errorf("analyzing MR %s: %w", mr.ID, err)
		}
		queueAnalysis.MRs = append(queueAnalysis.MRs, analysis)
	}

	// Calculate queue metrics
	queueAnalysis.TotalRisk = a.calculateTotalRisk(queueAnalysis.MRs)
	queueAnalysis.ConflictPairs = a.extractConflictPairs(queueAnalysis.MRs)
	queueAnalysis.RecommendedOrder = a.recommendMergeOrder(queueAnalysis.MRs)
	queueAnalysis.QueueHealth = a.assessQueueHealth(queueAnalysis.MRs)

	return queueAnalysis, nil
}

// analyzeConflictRisk analyzes conflict-related risks.
func (a *Analyzer) analyzeConflictRisk(mr *refinery.MRInfo, queue []*refinery.MRInfo) (ConflictRisk, []ConflictPrediction, error) {
	risk := ConflictRisk{Score: 0}
	var conflicts []ConflictPrediction

	if !a.config.IncludeConflicts {
		return risk, conflicts, nil
	}

	// Get changed files for this MR
	changedFiles, err := a.getChangedFiles(mr.Branch, mr.Target)
	if err != nil {
		return risk, conflicts, fmt.Errorf("getting changed files: %w", err)
	}

	// Check for overlapping MRs
	overlapping := 0
	for _, other := range queue {
		if other.ID == mr.ID {
			continue
		}

		otherFiles, err := a.getChangedFiles(other.Branch, other.Target)
		if err != nil {
			// Non-fatal: skip this MR
			continue
		}

		overlap := fileOverlap(changedFiles, otherFiles)
		if len(overlap) > 0 {
			overlapping++
			conflicts = append(conflicts, ConflictPrediction{
				WithMR:     other.ID,
				WithBranch: other.Branch,
				Files:      overlap,
				Severity:   a.assessConflictSeverity(len(overlap), otherFiles),
				Confidence: 0.8,
				Type:       ConflictDirect,
				Suggestions: []string{
					fmt.Sprintf("Coordinate with %s", other.Worker),
					fmt.Sprintf("Consider merging before/after %s", other.ID),
				},
			})
		}
	}

	risk.OverlappingMRs = overlapping
	risk.Score += overlapping * 10 // +10 per overlapping MR

	// Check target divergence
	divergence, err := a.getTargetDivergence(mr.Branch, mr.Target)
	if err == nil {
		risk.TargetDivergence = divergence
		if divergence > 10 {
			risk.Score += 5
		}
	}

	// Check merge base age
	baseAge, err := a.getMergeBaseAge(mr.Branch, mr.Target)
	if err == nil {
		risk.MergeBaseAge = baseAge
		if baseAge > 7 {
			risk.Score += 5
		}
	}

	// Cap conflict risk at 30
	if risk.Score > 30 {
		risk.Score = 30
	}

	risk.Details = fmt.Sprintf("%d overlapping MRs, %d commits divergence", overlapping, divergence)
	return risk, conflicts, nil
}

// analyzeTestRisk analyzes testing-related risks.
func (a *Analyzer) analyzeTestRisk(mr *refinery.MRInfo) (TestRisk, error) {
	risk := TestRisk{Score: 0}

	if !a.config.IncludeTesting {
		return risk, nil
	}

	// Check for test file changes
	changedFiles, err := a.getChangedFiles(mr.Branch, mr.Target)
	if err != nil {
		return risk, fmt.Errorf("getting changed files: %w", err)
	}

	hasTestFiles := false
	hasCodeFiles := false
	for _, file := range changedFiles {
		if isTestFile(file) {
			hasTestFiles = true
		} else if isCodeFile(file) {
			hasCodeFiles = true
		}
	}

	risk.HasTests = hasTestFiles

	// Penalize code changes without tests
	if hasCodeFiles && !hasTestFiles {
		risk.Score += 10
		risk.Details = "Code changes without test coverage"
	}

	// Cap test risk at 20
	if risk.Score > 20 {
		risk.Score = 20
	}

	return risk, nil
}

// analyzeSizeRisk analyzes size-related risks.
func (a *Analyzer) analyzeSizeRisk(mr *refinery.MRInfo) (SizeRisk, error) {
	risk := SizeRisk{Score: 0}

	// Get diff stats
	stats, err := a.getDiffStats(mr.Branch, mr.Target)
	if err != nil {
		return risk, fmt.Errorf("getting diff stats: %w", err)
	}

	risk.LinesChanged = stats.LinesAdded + stats.LinesDeleted
	risk.FilesChanged = stats.FilesChanged

	// Score based on lines changed
	switch {
	case risk.LinesChanged > 2000:
		risk.Score += 15
	case risk.LinesChanged > 1000:
		risk.Score += 10
	case risk.LinesChanged > 500:
		risk.Score += 5
	}

	// Score based on files changed
	switch {
	case risk.FilesChanged > 50:
		risk.Score += 10
	case risk.FilesChanged > 20:
		risk.Score += 5
	}

	// Cap size risk at 15
	if risk.Score > 15 {
		risk.Score = 15
	}

	risk.Details = fmt.Sprintf("%d lines, %d files", risk.LinesChanged, risk.FilesChanged)
	return risk, nil
}

// analyzeDependencyRisk analyzes dependency-related risks.
func (a *Analyzer) analyzeDependencyRisk(mr *refinery.MRInfo, queue []*refinery.MRInfo) (DependencyRisk, error) {
	risk := DependencyRisk{Score: 0, BlockedBy: make([]string, 0)}

	// Check if this MR blocks others
	for _, other := range queue {
		if other.ID == mr.ID {
			continue
		}
		if other.BlockedBy == mr.ID {
			risk.BlocksOthers++
		}
	}

	if risk.BlocksOthers > 0 {
		risk.Score += 5
	}

	// Check if this MR is blocked
	if mr.BlockedBy != "" {
		risk.BlockedBy = append(risk.BlockedBy, mr.BlockedBy)
		// Check if blocking MR is failing
		for _, other := range queue {
			if other.ID == mr.BlockedBy {
				// Assume failing if retry count > 0
				if other.RetryCount > 0 {
					risk.Score += 10
				}
				break
			}
		}
	}

	// Cap dependency risk at 10
	if risk.Score > 10 {
		risk.Score = 10
	}

	risk.Details = fmt.Sprintf("Blocks %d MRs, blocked by %d MRs", risk.BlocksOthers, len(risk.BlockedBy))
	return risk, nil
}

// analyzeHistoryRisk analyzes historical patterns.
func (a *Analyzer) analyzeHistoryRisk(mr *refinery.MRInfo) (HistoryRisk, error) {
	risk := HistoryRisk{Score: 0}

	// This is a placeholder for historical analysis
	// In a full implementation, this would:
	// 1. Query git history for author's merge success rate
	// 2. Analyze file change velocity
	// 3. Look for patterns in recent failures

	risk.Details = "Historical analysis not yet implemented"
	return risk, nil
}

// generateRecommendations creates actionable recommendations.
func (a *Analyzer) generateRecommendations(analysis *MRAnalysis) []Recommendation {
	var recs []Recommendation

	// Conflict recommendations
	if analysis.ConflictRisk.Score > 15 {
		for _, conflict := range analysis.Conflicts {
			recs = append(recs, Recommendation{
				Priority: 1,
				Category: CategoryConflict,
				Message:  fmt.Sprintf("Merge after %s (conflicts in %d files)", conflict.WithMR, len(conflict.Files)),
				Action:   fmt.Sprintf("Wait for %s to merge", conflict.WithMR),
			})
		}
	}

	// Test recommendations
	if !analysis.TestRisk.HasTests && analysis.SizeRisk.LinesChanged > 100 {
		recs = append(recs, Recommendation{
			Priority: 2,
			Category: CategoryTesting,
			Message:  "Consider adding tests for code changes",
			Action:   "Add test coverage before merging",
		})
	}

	// Size recommendations
	if analysis.SizeRisk.LinesChanged > 1000 {
		recs = append(recs, Recommendation{
			Priority: 2,
			Category: CategorySize,
			Message:  "Large changeset increases merge risk",
			Action:   "Consider splitting into smaller MRs",
		})
	}

	// Sort by priority
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Priority < recs[j].Priority
	})

	return recs
}

// determineOptimalWindow calculates the best time to merge.
func (a *Analyzer) determineOptimalWindow(mr *refinery.MRInfo, queue []*refinery.MRInfo, conflicts []ConflictPrediction) *MergeWindow {
	window := &MergeWindow{
		After:  make([]string, 0),
		Before: make([]string, 0),
	}

	// Add conflicting MRs to "after" list
	for _, conflict := range conflicts {
		if conflict.Confidence > a.config.ConflictThreshold {
			window.After = append(window.After, conflict.WithMR)
		}
	}

	// Estimate wait time (assume 1 hour per MR ahead in queue)
	window.EstimatedWait = time.Duration(len(window.After)) * time.Hour

	if len(window.After) > 0 {
		window.Reasoning = fmt.Sprintf("Wait for %d conflicting MR(s) to merge first", len(window.After))
	} else {
		window.Reasoning = "Safe to merge anytime"
	}

	return window
}

// Helper methods

func (a *Analyzer) getChangedFiles(branch, target string) ([]string, error) {
	// TODO: Implement using git diff --name-only
	// For now, return empty list as placeholder
	return []string{}, nil
}

func (a *Analyzer) getTargetDivergence(branch, target string) (int, error) {
	// Count commits on target not in branch using existing git package
	count, err := a.git.CommitsAhead(branch, target)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (a *Analyzer) getMergeBaseAge(branch, target string) (int, error) {
	// TODO: Implement using git merge-base and git log
	// For now, return 0 days as placeholder
	return 0, nil
}

// DiffStats represents diff statistics.
type DiffStats struct {
	FilesChanged int
	LinesAdded   int
	LinesDeleted int
}

func (a *Analyzer) getDiffStats(branch, target string) (*DiffStats, error) {
	// TODO: Implement using git diff --numstat
	// For now, return placeholder stats
	return &DiffStats{
		FilesChanged: 5,
		LinesAdded:   100,
		LinesDeleted: 50,
	}, nil
}

func (a *Analyzer) assessConflictSeverity(overlapCount int, totalFiles []string) ConflictSeverity {
	ratio := float64(overlapCount) / float64(len(totalFiles))
	switch {
	case ratio > 0.5:
		return SeverityHigh
	case ratio > 0.2:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

func (a *Analyzer) calculateTotalRisk(analyses []*MRAnalysis) int {
	total := 0
	for _, a := range analyses {
		total += a.RiskScore
	}
	if len(analyses) > 0 {
		return total / len(analyses)
	}
	return 0
}

func (a *Analyzer) extractConflictPairs(analyses []*MRAnalysis) []ConflictPair {
	pairs := make([]ConflictPair, 0)
	seen := make(map[string]bool)

	for _, analysis := range analyses {
		for _, conflict := range analysis.Conflicts {
			pairKey := analysis.MR.ID + "-" + conflict.WithMR
			reversePairKey := conflict.WithMR + "-" + analysis.MR.ID

			if !seen[pairKey] && !seen[reversePairKey] {
				pairs = append(pairs, ConflictPair{
					MR1:      analysis.MR.ID,
					MR2:      conflict.WithMR,
					Files:    conflict.Files,
					Severity: conflict.Severity,
				})
				seen[pairKey] = true
			}
		}
	}

	return pairs
}

func (a *Analyzer) recommendMergeOrder(analyses []*MRAnalysis) []string {
	// Sort by risk score (lowest first)
	sorted := make([]*MRAnalysis, len(analyses))
	copy(sorted, analyses)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RiskScore < sorted[j].RiskScore
	})

	order := make([]string, len(sorted))
	for i, a := range sorted {
		order[i] = a.MR.ID
	}
	return order
}

func (a *Analyzer) assessQueueHealth(analyses []*MRAnalysis) QueueHealth {
	health := QueueHealth{
		Recommendations: make([]string, 0),
	}

	highRisk := 0
	totalAge := time.Duration(0)

	for _, analysis := range analyses {
		if analysis.RiskLevel == RiskHigh || analysis.RiskLevel == RiskCritical {
			highRisk++
		}
		totalAge += time.Since(analysis.MR.CreatedAt)
	}

	health.HighRiskCount = highRisk
	health.ConflictCount = len(a.extractConflictPairs(analyses))

	if len(analyses) > 0 {
		health.AverageAge = totalAge / time.Duration(len(analyses))
	}

	// Determine status
	switch {
	case highRisk > 2 || health.ConflictCount > 3:
		health.Status = QueueCritical
		health.Recommendations = append(health.Recommendations, "Address high-risk MRs before continuing")
	case highRisk > 0 || health.ConflictCount > 1:
		health.Status = QueueWarning
		health.Recommendations = append(health.Recommendations, "Monitor conflicts and high-risk merges")
	default:
		health.Status = QueueHealthy
	}

	return health
}

// Utility functions

func fileOverlap(files1, files2 []string) []string {
	set := make(map[string]bool)
	for _, f := range files1 {
		set[f] = true
	}

	overlap := make([]string, 0)
	for _, f := range files2 {
		if set[f] {
			overlap = append(overlap, f)
		}
	}
	return overlap
}

func isTestFile(path string) bool {
	return len(path) > 8 && (path[len(path)-8:] == "_test.go" ||
		path[len(path)-8:] == ".test.ts" ||
		path[len(path)-8:] == ".test.js" ||
		(len(path) > 5 && path[:5] == "test_"))
}

func isCodeFile(path string) bool {
	// Simple heuristic: files with code extensions
	for _, ext := range []string{".go", ".ts", ".js", ".py", ".java", ".c", ".cpp"} {
		if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	return false
}
