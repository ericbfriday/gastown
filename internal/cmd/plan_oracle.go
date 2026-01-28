// Package cmd implements CLI commands.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/planoracle/cmd"
	"github.com/steveyegge/gastown/internal/planoracle/sources"
	"github.com/steveyegge/gastown/internal/runtime"
)

func init() {
	rootCmd.AddCommand(planOracleCmd)
}

var planOracleCmd = &cobra.Command{
	Use:   "plan-oracle",
	Short: "Intelligent work planning and analysis",
	Long: `Plan-oracle provides intelligent analysis of work items and planning documents.

Features:
  - Work decomposition (break down epics into tasks)
  - Dependency analysis and visualization
  - Effort estimation based on historical patterns
  - Execution ordering recommendations
  - Risk identification and mitigation suggestions

Examples:
  # Decompose an epic into tasks
  gt plan-oracle decompose gt-35x

  # Analyze dependencies and complexity
  gt plan-oracle analyze gt-35x

  # Recommend execution order
  gt plan-oracle order gt-35x

  # Estimate effort for a work item
  gt plan-oracle estimate gt-35x

  # Visualize dependency graph
  gt plan-oracle visualize gt-35x`,
	Run: func(c *cobra.Command, args []string) {
		if err := c.Help(); err != nil {
			fmt.Fprintf(c.ErrOrStderr(), "Error: %v\n", err)
		}
	},
}

func init() {
	// Create beads source for plan-oracle commands
	workDir := runtime.GetWorkDir()
	beadsSource := sources.NewBeadsSource(workDir)

	// Register subcommands
	planOracleCmd.AddCommand(cmd.NewDecomposeCmd(beadsSource))
	planOracleCmd.AddCommand(cmd.NewAnalyzeCmd(beadsSource))
	planOracleCmd.AddCommand(cmd.NewOrderCmd(beadsSource))
	planOracleCmd.AddCommand(cmd.NewEstimateCmd(beadsSource))
	planOracleCmd.AddCommand(cmd.NewVisualizeCmd(beadsSource))
}
