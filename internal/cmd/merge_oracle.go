package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/constants"
	"github.com/steveyegge/gastown/internal/mergeoracle"
	"github.com/steveyegge/gastown/internal/refinery"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

// Merge oracle command flags
var (
	mergeOracleJSON       bool
	mergeOracleRig        string
	mergeOracleVerbose    bool
	mergeOracleNoConflict bool
	mergeOracleNoTest     bool
	mergeOracleNoHistory  bool
)

var mergeOracleCmd = &cobra.Command{
	Use:     "merge-oracle",
	GroupID: GroupAnalysis,
	Short:   "Intelligent merge queue analysis",
	Long: `Analyze merge queue for conflicts, risks, and optimal merge timing.

The merge-oracle analyzes pending merge requests to:
  - Predict merge conflicts before they happen
  - Assess risk based on size, testing, and dependencies
  - Recommend optimal merge order
  - Suggest best merge timing

This is an advisory tool - it never blocks merges, only provides insights.

Examples:
  gt merge-oracle queue                # Show queue with risk scores
  gt merge-oracle analyze <branch>     # Analyze specific branch
  gt merge-oracle conflicts            # Show potential conflicts
  gt merge-oracle recommend            # Recommend merge order`,
	RunE: requireSubcommand,
}

var mergeOracleQueueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Show merge queue with risk analysis",
	Long: `Display the merge queue with risk scores and conflict warnings.

Shows all pending merge requests with:
  - Risk score (0-100, lower is safer)
  - Risk level (low/medium/high/critical)
  - Potential conflicts
  - Age and basic stats

Examples:
  gt merge-oracle queue                # Human-readable output
  gt merge-oracle queue --json         # JSON output
  gt merge-oracle queue --rig aardwolf # Specific rig`,
	RunE: runMergeOracleQueue,
}

var mergeOracleAnalyzeCmd = &cobra.Command{
	Use:   "analyze <branch>",
	Short: "Analyze merge safety for a branch",
	Long: `Perform comprehensive analysis of a branch's merge readiness.

Provides detailed risk assessment including:
  - Overall risk score and level
  - Conflict predictions with specific MRs
  - Test coverage analysis
  - Size and complexity assessment
  - Dependency analysis
  - Actionable recommendations
  - Optimal merge window

Examples:
  gt merge-oracle analyze feature/new-api
  gt merge-oracle analyze polecat/nux --verbose
  gt merge-oracle analyze mybranch --json`,
	Args: cobra.ExactArgs(1),
	RunE: runMergeOracleAnalyze,
}

var mergeOracleConflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "Predict potential merge conflicts",
	Long: `Analyze the merge queue for potential conflicts between pending MRs.

Shows:
  - All MR pairs with overlapping file changes
  - Conflict severity (low/medium/high)
  - Specific files that may conflict
  - Suggestions to avoid conflicts

Examples:
  gt merge-oracle conflicts
  gt merge-oracle conflicts --json
  gt merge-oracle conflicts --rig duneagent`,
	RunE: runMergeOracleConflicts,
}

var mergeOracleRecommendCmd = &cobra.Command{
	Use:   "recommend",
	Short: "Recommend optimal merge order",
	Long: `Recommend the optimal order to merge pending MRs.

Considers:
  - Risk scores (merge safer MRs first)
  - Conflicts (avoid conflicting merges in sequence)
  - Dependencies (respect blocking relationships)
  - Priority (P0 tasks first)

Examples:
  gt merge-oracle recommend
  gt merge-oracle recommend --json`,
	RunE: runMergeOracleRecommend,
}

func init() {
	// Queue subcommand flags
	mergeOracleQueueCmd.Flags().BoolVar(&mergeOracleJSON, "json", false, "Output as JSON")
	mergeOracleQueueCmd.Flags().StringVar(&mergeOracleRig, "rig", "", "Analyze specific rig (default: current or all)")
	mergeOracleQueueCmd.Flags().BoolVarP(&mergeOracleVerbose, "verbose", "v", false, "Show detailed information")

	// Analyze subcommand flags
	mergeOracleAnalyzeCmd.Flags().BoolVar(&mergeOracleJSON, "json", false, "Output as JSON")
	mergeOracleAnalyzeCmd.Flags().BoolVarP(&mergeOracleVerbose, "verbose", "v", false, "Show detailed analysis")
	mergeOracleAnalyzeCmd.Flags().BoolVar(&mergeOracleNoConflict, "no-conflict", false, "Skip conflict analysis")
	mergeOracleAnalyzeCmd.Flags().BoolVar(&mergeOracleNoTest, "no-test", false, "Skip test analysis")
	mergeOracleAnalyzeCmd.Flags().BoolVar(&mergeOracleNoHistory, "no-history", false, "Skip historical analysis")

	// Conflicts subcommand flags
	mergeOracleConflictsCmd.Flags().BoolVar(&mergeOracleJSON, "json", false, "Output as JSON")
	mergeOracleConflictsCmd.Flags().StringVar(&mergeOracleRig, "rig", "", "Analyze specific rig")

	// Recommend subcommand flags
	mergeOracleRecommendCmd.Flags().BoolVar(&mergeOracleJSON, "json", false, "Output as JSON")
	mergeOracleRecommendCmd.Flags().StringVar(&mergeOracleRig, "rig", "", "Analyze specific rig")

	// Add subcommands
	mergeOracleCmd.AddCommand(mergeOracleQueueCmd)
	mergeOracleCmd.AddCommand(mergeOracleAnalyzeCmd)
	mergeOracleCmd.AddCommand(mergeOracleConflictsCmd)
	mergeOracleCmd.AddCommand(mergeOracleRecommendCmd)

	rootCmd.AddCommand(mergeOracleCmd)
}

func runMergeOracleQueue(cmd *cobra.Command, args []string) error {
	townRoot, r, b, err := setupMergeOracle()
	if err != nil {
		return err
	}

	// Get merge queue
	queue, err := getMergeQueue(b, r)
	if err != nil {
		return err
	}

	if len(queue) == 0 {
		fmt.Printf("%s Merge queue is empty\n", style.Dim.Render("○"))
		return nil
	}

	// Analyze queue
	analyzer, err := mergeoracle.NewAnalyzer(r.Path, getAnalysisConfig())
	if err != nil {
		return fmt.Errorf("creating analyzer: %w", err)
	}

	queueAnalysis, err := analyzer.AnalyzeQueue(queue)
	if err != nil {
		return fmt.Errorf("analyzing queue: %w", err)
	}

	if mergeOracleJSON {
		return outputJSON(queueAnalysis)
	}

	return outputQueueText(queueAnalysis, townRoot)
}

func runMergeOracleAnalyze(cmd *cobra.Command, args []string) error {
	branch := args[0]

	townRoot, r, b, err := setupMergeOracle()
	if err != nil {
		return err
	}

	// Get merge queue
	queue, err := getMergeQueue(b, r)
	if err != nil {
		return err
	}

	// Find the MR for this branch
	var targetMR *refinery.MRInfo
	for _, mr := range queue {
		if mr.Branch == branch || strings.HasSuffix(mr.Branch, "/"+branch) {
			targetMR = mr
			break
		}
	}

	if targetMR == nil {
		return fmt.Errorf("branch %q not found in merge queue", branch)
	}

	// Analyze MR
	analyzer, err := mergeoracle.NewAnalyzer(r.Path, getAnalysisConfig())
	if err != nil {
		return fmt.Errorf("creating analyzer: %w", err)
	}

	analysis, err := analyzer.AnalyzeMR(targetMR, queue)
	if err != nil {
		return fmt.Errorf("analyzing MR: %w", err)
	}

	if mergeOracleJSON {
		return outputJSON(analysis)
	}

	return outputAnalysisText(analysis, townRoot)
}

func runMergeOracleConflicts(cmd *cobra.Command, args []string) error {
	_, r, b, err := setupMergeOracle()
	if err != nil {
		return err
	}

	// Get merge queue
	queue, err := getMergeQueue(b, r)
	if err != nil {
		return err
	}

	if len(queue) == 0 {
		fmt.Printf("%s No pending merge requests\n", style.Dim.Render("○"))
		return nil
	}

	// Analyze queue for conflicts
	analyzer, err := mergeoracle.NewAnalyzer(r.Path, getAnalysisConfig())
	if err != nil {
		return fmt.Errorf("creating analyzer: %w", err)
	}

	queueAnalysis, err := analyzer.AnalyzeQueue(queue)
	if err != nil {
		return fmt.Errorf("analyzing queue: %w", err)
	}

	if mergeOracleJSON {
		return outputJSON(queueAnalysis.ConflictPairs)
	}

	return outputConflictsText(queueAnalysis.ConflictPairs)
}

func runMergeOracleRecommend(cmd *cobra.Command, args []string) error {
	_, r, b, err := setupMergeOracle()
	if err != nil {
		return err
	}

	// Get merge queue
	queue, err := getMergeQueue(b, r)
	if err != nil {
		return err
	}

	if len(queue) == 0 {
		fmt.Printf("%s No pending merge requests\n", style.Dim.Render("○"))
		return nil
	}

	// Analyze queue
	analyzer, err := mergeoracle.NewAnalyzer(r.Path, getAnalysisConfig())
	if err != nil {
		return fmt.Errorf("creating analyzer: %w", err)
	}

	queueAnalysis, err := analyzer.AnalyzeQueue(queue)
	if err != nil {
		return fmt.Errorf("analyzing queue: %w", err)
	}

	if mergeOracleJSON {
		return outputJSON(queueAnalysis.RecommendedOrder)
	}

	return outputRecommendationsText(queueAnalysis)
}

// Helper functions

func setupMergeOracle() (string, *rig.Rig, *beads.Beads, error) {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return "", nil, nil, fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Determine which rig to analyze
	var r *rig.Rig
	if mergeOracleRig != "" {
		// Load specific rig
		rigsConfigPath := constants.MayorRigsPath(townRoot)
		rigsConfig, err := config.LoadRigsConfig(rigsConfigPath)
		if err != nil {
			return "", nil, nil, fmt.Errorf("loading rigs config: %w", err)
		}

		rigEntry, ok := rigsConfig.Rigs[mergeOracleRig]
		if !ok {
			return "", nil, nil, fmt.Errorf("rig %q not found", mergeOracleRig)
		}

		r, err = rig.Load(rigEntry.Path)
		if err != nil {
			return "", nil, nil, fmt.Errorf("loading rig: %w", err)
		}
	} else {
		// Try to detect current rig from cwd
		var err error
		r, err = rig.FindFromCwd()
		if err != nil {
			return "", nil, nil, fmt.Errorf("not in a rig (specify --rig): %w", err)
		}
	}

	// Initialize beads
	b, err := beads.New(r.Path)
	if err != nil {
		return "", nil, nil, fmt.Errorf("initializing beads: %w", err)
	}

	return townRoot, r, b, nil
}

func getMergeQueue(b *beads.Beads, r *rig.Rig) ([]*refinery.MRInfo, error) {
	// Query beads for merge-request issues
	// This is a simplified version - in reality, we'd use the refinery package
	// to properly query the merge queue

	// For now, return empty queue as placeholder
	// TODO: Implement actual merge queue query
	return []*refinery.MRInfo{}, nil
}

func getAnalysisConfig() *mergeoracle.AnalysisConfig {
	config := mergeoracle.DefaultAnalysisConfig()
	config.IncludeConflicts = !mergeOracleNoConflict
	config.IncludeTesting = !mergeOracleNoTest
	config.IncludeHistory = !mergeOracleNoHistory
	return config
}

func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func outputQueueText(qa *mergeoracle.QueueAnalysis, townRoot string) error {
	fmt.Printf("%s Merge Queue Analysis\n\n", style.Bold.Render("●"))

	// Queue health
	healthIcon := "🟢"
	switch qa.QueueHealth.Status {
	case mergeoracle.QueueWarning:
		healthIcon = "🟡"
	case mergeoracle.QueueCritical:
		healthIcon = "🔴"
	}
	fmt.Printf("Queue Health: %s %s\n", healthIcon, qa.QueueHealth.Status)
	fmt.Printf("Total MRs: %d\n", len(qa.MRs))
	fmt.Printf("High Risk: %d\n", qa.QueueHealth.HighRiskCount)
	fmt.Printf("Conflicts: %d\n", qa.QueueHealth.ConflictCount)
	fmt.Printf("Average Age: %s\n\n", formatDuration(qa.QueueHealth.AverageAge))

	// MR table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Pos\tMR\tRisk\tBranch\tFiles\tAge\t")
	fmt.Fprintln(w, "---\t---\t----\t------\t-----\t---\t")

	for i, analysis := range qa.MRs {
		age := time.Since(analysis.MR.CreatedAt)
		risk := fmt.Sprintf("%s %d", analysis.RiskLevel.Icon(), analysis.RiskScore)
		branch := analysis.MR.Branch
		if len(branch) > 20 {
			branch = branch[:17] + "..."
		}

		warning := ""
		if len(analysis.Conflicts) > 0 {
			warning = "⚠"
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%s\t%s\n",
			i+1,
			analysis.MR.ID,
			risk,
			branch,
			analysis.SizeRisk.FilesChanged,
			formatDuration(age),
			warning)
	}
	w.Flush()

	// Recommendations
	if len(qa.QueueHealth.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for _, rec := range qa.QueueHealth.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
	}

	return nil
}

func outputAnalysisText(analysis *mergeoracle.MRAnalysis, townRoot string) error {
	fmt.Printf("%s Merge Analysis: %s\n\n", style.Bold.Render("●"), analysis.MR.Branch)

	// Risk score
	fmt.Printf("Risk Score: %s %d/100 (%s)\n\n",
		analysis.RiskLevel.Icon(),
		analysis.RiskScore,
		analysis.RiskLevel.String())

	// Risk breakdown
	fmt.Println("Risk Breakdown:")
	printRiskComponent("Conflict Risk", analysis.ConflictRisk.Score, 30, analysis.ConflictRisk.Details)
	printRiskComponent("Test Risk", analysis.TestRisk.Score, 20, analysis.TestRisk.Details)
	printRiskComponent("Size Risk", analysis.SizeRisk.Score, 15, analysis.SizeRisk.Details)
	printRiskComponent("Dependency Risk", analysis.DependencyRisk.Score, 10, analysis.DependencyRisk.Details)
	printRiskComponent("History Risk", analysis.HistoryRisk.Score, 5, analysis.HistoryRisk.Details)

	// Conflicts
	if len(analysis.Conflicts) > 0 {
		fmt.Println("\nPotential Conflicts:")
		for _, conflict := range analysis.Conflicts {
			fmt.Printf("  %s %s: %d files (severity: %s, confidence: %.0f%%)\n",
				"⚠",
				conflict.WithMR,
				len(conflict.Files),
				conflict.Severity,
				conflict.Confidence*100)
			if mergeOracleVerbose {
				for _, file := range conflict.Files {
					fmt.Printf("    - %s\n", file)
				}
			}
		}
	}

	// Recommendations
	if len(analysis.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for i, rec := range analysis.Recommendations {
			fmt.Printf("  %d. [%s] %s\n", i+1, rec.Category, rec.Message)
			if rec.Action != "" && mergeOracleVerbose {
				fmt.Printf("     Action: %s\n", rec.Action)
			}
		}
	}

	// Optimal window
	if analysis.OptimalWindow != nil {
		fmt.Println("\nOptimal Merge Window:")
		fmt.Printf("  %s\n", analysis.OptimalWindow.Reasoning)
		if len(analysis.OptimalWindow.After) > 0 {
			fmt.Printf("  After: %s\n", strings.Join(analysis.OptimalWindow.After, ", "))
		}
		if analysis.OptimalWindow.EstimatedWait > 0 {
			fmt.Printf("  Estimated wait: %s\n", formatDuration(analysis.OptimalWindow.EstimatedWait))
		}
	}

	return nil
}

func outputConflictsText(pairs []mergeoracle.ConflictPair) error {
	if len(pairs) == 0 {
		fmt.Printf("%s No conflicts predicted\n", style.Success.Render("✓"))
		return nil
	}

	fmt.Printf("%s Potential Conflicts (%d pairs)\n\n", style.Warning.Render("⚠"), len(pairs))

	for _, pair := range pairs {
		icon := "⚠"
		if pair.Severity == mergeoracle.SeverityHigh {
			icon = "🔴"
		}
		fmt.Printf("%s %s ↔ %s (%d files, severity: %s)\n",
			icon, pair.MR1, pair.MR2, len(pair.Files), pair.Severity)

		if mergeOracleVerbose {
			for _, file := range pair.Files {
				fmt.Printf("    - %s\n", file)
			}
		}
	}

	return nil
}

func outputRecommendationsText(qa *mergeoracle.QueueAnalysis) error {
	fmt.Printf("%s Recommended Merge Order\n\n", style.Bold.Render("●"))

	for i, mrID := range qa.RecommendedOrder {
		// Find analysis for this MR
		var analysis *mergeoracle.MRAnalysis
		for _, a := range qa.MRs {
			if a.MR.ID == mrID {
				analysis = a
				break
			}
		}

		if analysis == nil {
			continue
		}

		fmt.Printf("%d. %s %s (risk: %s %d)\n",
			i+1,
			mrID,
			analysis.MR.Branch,
			analysis.RiskLevel.Icon(),
			analysis.RiskScore)
	}

	return nil
}

func printRiskComponent(name string, score int, max int, details string) {
	icon := "🟢"
	if score > max*2/3 {
		icon = "🔴"
	} else if score > max/3 {
		icon = "🟡"
	}

	fmt.Printf("  %s %s: %d/%d pts", icon, name, score, max)
	if details != "" {
		fmt.Printf(" (%s)", style.Dim.Render(details))
	}
	fmt.Println()
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
