package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/workspace/cleanup"
)

var (
	cleanDryRun       bool
	cleanVerbose      bool
	cleanJSON         bool
	cleanWorkspaceType string
	cleanPreflight    bool
	cleanPostflight   bool
)

func newWorkspaceCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [path]",
		Short: "Clean workspace (remove temp files, clear caches)",
		Long: `Clean workspace by removing temporary files, build caches, and other artifacts.

By default, runs postflight cleanup. Use --preflight to run preflight checks instead.

Examples:
  # Clean current workspace (postflight)
  gt workspace clean

  # Dry-run to preview changes
  gt workspace clean --dry-run

  # Run preflight checks before starting work
  gt workspace clean --preflight

  # Clean specific workspace type
  gt workspace clean --type crew

  # Clean with verbose output
  gt workspace clean --verbose
`,
		RunE: runWorkspaceClean,
	}

	cmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Preview changes without applying")
	cmd.Flags().BoolVarP(&cleanVerbose, "verbose", "v", false, "Verbose output")
	cmd.Flags().BoolVar(&cleanJSON, "json", false, "Output results as JSON")
	cmd.Flags().StringVarP(&cleanWorkspaceType, "type", "t", "crew", "Workspace type (crew|polecat|mayor|refinery|town)")
	cmd.Flags().BoolVar(&cleanPreflight, "preflight", false, "Run preflight checks")
	cmd.Flags().BoolVar(&cleanPostflight, "postflight", true, "Run postflight cleanup")

	return cmd
}

func runWorkspaceClean(cmd *cobra.Command, args []string) error {
	// Determine working directory
	workingDir := "."
	if len(args) > 0 {
		workingDir = args[0]
	}

	// Convert workspace type string to type
	var wsType cleanup.WorkspaceType
	switch cleanWorkspaceType {
	case "crew":
		wsType = cleanup.WorkspaceTypeCrew
	case "polecat":
		wsType = cleanup.WorkspaceTypePolecat
	case "mayor":
		wsType = cleanup.WorkspaceTypeMayor
	case "refinery":
		wsType = cleanup.WorkspaceTypeRefinery
	case "town":
		wsType = cleanup.WorkspaceTypeTown
	default:
		return fmt.Errorf("unknown workspace type: %s", cleanWorkspaceType)
	}

	// Load configuration
	config, err := cleanup.LoadConfig(workingDir, wsType)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create cleaner
	cleaner := cleanup.NewCleaner(workingDir, config)
	cleaner.SetDryRun(cleanDryRun)
	cleaner.SetVerbose(cleanVerbose)

	// Run cleanup
	var results []*cleanup.CleanupResult

	if cleanPreflight {
		if !cleanJSON {
			fmt.Printf("Running preflight checks in %s...\n", workingDir)
		}
		preResults, err := cleaner.RunPreflight()
		if err != nil {
			return fmt.Errorf("preflight failed: %w", err)
		}
		results = append(results, preResults...)
	} else if cleanPostflight {
		if !cleanJSON {
			fmt.Printf("Running postflight cleanup in %s...\n", workingDir)
		}
		postResults, err := cleaner.RunPostflight()
		if err != nil {
			return fmt.Errorf("postflight failed: %w", err)
		}
		results = append(results, postResults...)
	}

	// Output results
	if cleanJSON {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	if cleanDryRun {
		fmt.Println("\n🔍 DRY RUN - No changes will be made\n")
	}

	totalFiles := 0
	totalBytes := int64(0)
	hadErrors := false

	for _, result := range results {
		if result.Success {
			fmt.Printf("✅ %s\n", result.Action)
		} else {
			fmt.Printf("❌ %s\n", result.Action)
			hadErrors = true
		}

		if cleanVerbose {
			fmt.Printf("   Duration: %s\n", result.Duration)
			fmt.Printf("   Files found: %d\n", result.FilesFound)
			if !cleanDryRun {
				fmt.Printf("   Files removed: %d\n", result.FilesRemoved)
				if result.BytesFreed > 0 {
					fmt.Printf("   Space freed: %.2f MB\n", float64(result.BytesFreed)/(1024*1024))
				}
			}
		}

		if len(result.Errors) > 0 {
			for _, errMsg := range result.Errors {
				fmt.Printf("   ⚠️  %s\n", errMsg)
			}
		}

		if cleanVerbose && len(result.Details) > 0 {
			fmt.Printf("   Files:\n")
			for i, detail := range result.Details {
				if i >= 10 {
					fmt.Printf("   ... and %d more\n", len(result.Details)-10)
					break
				}
				fmt.Printf("     %s %s\n", detail.Action, detail.Path)
			}
		}

		totalFiles += result.FilesRemoved
		totalBytes += result.BytesFreed
		fmt.Println()
	}

	// Summary
	if !cleanDryRun && totalFiles > 0 {
		fmt.Printf("Summary: Removed %d files, freed %.2f MB\n", totalFiles, float64(totalBytes)/(1024*1024))
	}

	if hadErrors {
		return fmt.Errorf("cleanup completed with errors")
	}

	return nil
}
