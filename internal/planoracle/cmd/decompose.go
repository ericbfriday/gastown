// Package cmd implements plan-oracle CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/planoracle/analyzer"
	"github.com/steveyegge/gastown/internal/planoracle/sources"
)

// DecomposeOptions holds options for the decompose command.
type DecomposeOptions struct {
	AutoCreate   bool
	EstimateOnly bool
	Template     string
	Format       string
}

// NewDecomposeCmd creates the decompose subcommand.
func NewDecomposeCmd(beadsSource *sources.BeadsSource) *cobra.Command {
	opts := &DecomposeOptions{}

	cmd := &cobra.Command{
		Use:   "decompose <issue-id>",
		Short: "Break down an epic or feature into tasks",
		Long: `Decompose analyzes a large work item and breaks it down into smaller, actionable tasks.

The decomposition strategy depends on the work item's description:
1. Markdown task lists (- [ ] items)
2. Numbered steps or phases
3. Type-based templates (epic, feature, convoy)
4. Historical patterns from similar work

Examples:
  # Show decomposition without creating tasks
  gt plan-oracle decompose gt-35x

  # Auto-create tasks in beads
  gt plan-oracle decompose gt-35x --auto-create

  # Only show estimates
  gt plan-oracle decompose gt-35x --estimate-only

  # Use specific template
  gt plan-oracle decompose gt-35x --template plugin`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDecompose(beadsSource, args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.AutoCreate, "auto-create", false, "Automatically create child tasks")
	cmd.Flags().BoolVar(&opts.EstimateOnly, "estimate-only", false, "Show estimates only")
	cmd.Flags().StringVar(&opts.Template, "template", "", "Use specific template (plugin, feature, epic)")
	cmd.Flags().StringVar(&opts.Format, "format", "text", "Output format (text, json)")

	return cmd
}

func runDecompose(beadsSource *sources.BeadsSource, issueID string, opts *DecomposeOptions) error {
	// Load work item
	item, err := beadsSource.LoadWorkItem(issueID)
	if err != nil {
		return fmt.Errorf("failed to load work item: %w", err)
	}

	// Collect metrics for better decomposition
	metricsCollector := sources.NewMetricsCollector(beadsSource)
	metrics, err := metricsCollector.CollectMetrics()
	if err != nil {
		// Non-fatal: continue with empty metrics
		fmt.Fprintf(os.Stderr, "Warning: Could not collect historical metrics: %v\n", err)
		metrics = sources.NewMetricsCollector(beadsSource).CollectMetrics()
	}

	// Create decomposer
	decomposer := analyzer.NewDecomposer(metrics)

	// Decompose work item
	result, err := decomposer.Decompose(item)
	if err != nil {
		return fmt.Errorf("decomposition failed: %w", err)
	}

	// Display results
	if opts.Format == "json" {
		return displayDecompositionJSON(result)
	}

	return displayDecomposition(item, result, opts)
}

func displayDecomposition(item interface{}, result interface{}, opts *DecomposeOptions) error {
	// TODO: Implement text display
	fmt.Printf("Epic: %s\n", "item.ID")
	fmt.Printf("Total subtasks: %d\n", 0)
	fmt.Printf("Total estimate: %.1f days\n", 0.0)

	if !opts.EstimateOnly {
		fmt.Println("\nSubtasks:")
		// Display each subtask
	}

	if opts.AutoCreate {
		fmt.Println("\nCreating tasks in beads...")
		// Create tasks
	}

	return nil
}

func displayDecompositionJSON(result interface{}) error {
	// TODO: Implement JSON output
	return nil
}
