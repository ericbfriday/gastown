package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/planconvert"
	"github.com/steveyegge/gastown/internal/style"
)

var planToEpicCmd = &cobra.Command{
	Use:   "plan-to-epic <plan-file>",
	Short: "Convert planning document to beads epic",
	Long: `Convert a markdown planning document into a beads epic with subtasks.

This tool parses structured planning documents (like design docs with phases
and task lists) and generates beads-compatible epic structures.

Supported document formats:
  - Phase-based plans (Phase 1, Phase 2, etc.)
  - Task lists with numbered items
  - Deliverables with checkboxes
  - Success criteria sections

Output formats:
  - jsonl: Beads-compatible JSONL format (default)
  - json: Formatted JSON for inspection
  - pretty: Human-readable summary
  - shell: Shell script with bd commands

Examples:
  # Preview conversion
  gt plan-to-epic docs/design.md --dry-run

  # Generate JSONL output
  gt plan-to-epic docs/design.md --output epic.jsonl

  # Generate shell script for manual review
  gt plan-to-epic docs/design.md --format shell --output create-epic.sh

  # Create beads directly
  gt plan-to-epic docs/design.md --create --rig gastown`,
	Args: cobra.ExactArgs(1),
	RunE: runPlanToEpic,
}

var planToEpicOpts struct {
	output   string
	format   string
	prefix   string
	priority int
	dryRun   bool
	create   bool
	rig      string
}

func init() {
	planToEpicCmd.GroupID = GroupWork
	rootCmd.AddCommand(planToEpicCmd)

	planToEpicCmd.Flags().StringVarP(&planToEpicOpts.output, "output", "o", "",
		"Output file (default: stdout)")
	planToEpicCmd.Flags().StringVarP(&planToEpicOpts.format, "format", "f", "jsonl",
		"Output format: jsonl, json, pretty, shell")
	planToEpicCmd.Flags().StringVar(&planToEpicOpts.prefix, "prefix", "",
		"Epic ID prefix (default: auto-detect from filename)")
	planToEpicCmd.Flags().IntVarP(&planToEpicOpts.priority, "priority", "p", 2,
		"Default priority for tasks (1-4)")
	planToEpicCmd.Flags().BoolVar(&planToEpicOpts.dryRun, "dry-run", false,
		"Preview without creating output")
	planToEpicCmd.Flags().BoolVar(&planToEpicOpts.create, "create", false,
		"Create beads directly via bd CLI (requires --rig)")
	planToEpicCmd.Flags().StringVar(&planToEpicOpts.rig, "rig", "",
		"Target rig for bead creation (required with --create)")
}

func runPlanToEpic(cmd *cobra.Command, args []string) error {
	planFile := args[0]

	// Validate file exists
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", planFile)
	}

	// Validate file is markdown
	if !strings.HasSuffix(planFile, ".md") {
		return fmt.Errorf("file must be a markdown file (.md): %s", planFile)
	}

	// Validate flags
	if planToEpicOpts.create && planToEpicOpts.rig == "" {
		return fmt.Errorf("--rig is required when using --create")
	}

	if planToEpicOpts.priority < 1 || planToEpicOpts.priority > 4 {
		return fmt.Errorf("priority must be between 1 and 4")
	}

	// Auto-detect prefix from filename if not provided
	if planToEpicOpts.prefix == "" {
		base := filepath.Base(planFile)
		base = strings.TrimSuffix(base, ".md")
		// Convert to kebab-case and take first part
		parts := strings.Split(base, "-")
		if len(parts) > 0 {
			planToEpicOpts.prefix = parts[0]
		} else {
			planToEpicOpts.prefix = "plan"
		}
	}

	// Parse the planning document
	fmt.Fprintf(os.Stderr, "%s Parsing planning document: %s\n",
		style.ArrowPrefix, planFile)

	doc, err := planconvert.ParsePlanDocument(planFile)
	if err != nil {
		return fmt.Errorf("failed to parse document: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s Found: %s\n",
		style.SuccessPrefix, doc.Title)

	// Convert to epic
	opts := planconvert.ConversionOptions{
		Prefix:      planToEpicOpts.prefix,
		Priority:    planToEpicOpts.priority,
		DryRun:      planToEpicOpts.dryRun,
		OutputFile:  planToEpicOpts.output,
		CreateBeads: planToEpicOpts.create,
		TargetRig:   planToEpicOpts.rig,
	}

	epic, err := planconvert.ConvertToEpic(doc, opts)
	if err != nil {
		return fmt.Errorf("failed to convert to epic: %w", err)
	}

	// Print statistics
	fmt.Fprintf(os.Stderr, "%s Generated epic with %d tasks\n",
		style.SuccessPrefix, len(epic.Subtasks))

	// Dry run mode - just show summary
	if planToEpicOpts.dryRun {
		fmt.Fprintf(os.Stderr, "\n%s DRY RUN MODE - Preview:\n\n",
			style.WarningPrefix)

		err := planconvert.WriteEpic(epic, os.Stdout, planconvert.FormatPretty)
		if err != nil {
			return fmt.Errorf("failed to write preview: %w", err)
		}

		fmt.Fprintf(os.Stderr, "\n%s Run without --dry-run to generate output\n",
			style.ArrowPrefix)
		return nil
	}

	// Determine output format
	var format planconvert.OutputFormat
	switch planToEpicOpts.format {
	case "jsonl":
		format = planconvert.FormatJSONL
	case "json":
		format = planconvert.FormatJSON
	case "pretty":
		format = planconvert.FormatPretty
	case "shell":
		format = planconvert.FormatBeadsShell
	default:
		return fmt.Errorf("unsupported format: %s", planToEpicOpts.format)
	}

	// Write output
	if planToEpicOpts.output != "" {
		// Write to file
		err := planconvert.SaveToFile(epic, planToEpicOpts.output, format)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}

		fmt.Fprintf(os.Stderr, "%s Wrote output to: %s\n",
			style.SuccessPrefix, planToEpicOpts.output)

		if format == planconvert.FormatBeadsShell {
			fmt.Fprintf(os.Stderr, "%s Run: bash %s\n",
				style.ArrowPrefix, planToEpicOpts.output)
		}
	} else {
		// Write to stdout
		err := planconvert.WriteEpic(epic, os.Stdout, format)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	// Create beads if requested
	if planToEpicOpts.create {
		return createBeadsFromEpic(epic, planToEpicOpts.rig)
	}

	return nil
}

// createBeadsFromEpic creates beads using the bd CLI.
func createBeadsFromEpic(epic *planconvert.Epic, rig string) error {
	fmt.Fprintf(os.Stderr, "\n%s Creating beads in rig: %s\n",
		style.ArrowPrefix, rig)

	// TODO: Implement bd CLI integration
	// For now, just show what would be created

	fmt.Fprintf(os.Stderr, "%s Bead creation not yet implemented\n",
		style.WarningPrefix)
	fmt.Fprintf(os.Stderr, "%s Generate shell script with --format shell instead\n",
		style.ArrowPrefix)

	return fmt.Errorf("bead creation via CLI not yet implemented")
}
