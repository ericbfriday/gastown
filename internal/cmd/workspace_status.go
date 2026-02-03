package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/workspace/cleanup"
)

var (
	workspaceStatusJSON         bool
	workspaceStatusVerbose      bool
	statusWorkspaceType string
)

func newWorkspaceStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [path]",
		Short: "Show workspace status and state",
		Long: `Display the current state of a workspace including git status, temp files, and cleanliness.

Examples:
  # Show status of current workspace
  gt workspace status

  # Show status with details
  gt workspace status --verbose

  # Show status as JSON
  gt workspace status --json

  # Show status for specific workspace type
  gt workspace status --type crew
`,
		RunE: runWorkspaceStatus,
	}

	cmd.Flags().BoolVar(&workspaceStatusJSON, "json", false, "Output as JSON")
	cmd.Flags().BoolVarP(&workspaceStatusVerbose, "verbose", "v", false, "Verbose output")
	cmd.Flags().StringVarP(&statusWorkspaceType, "type", "t", "crew", "Workspace type")

	return cmd
}

func runWorkspaceStatus(cmd *cobra.Command, args []string) error {
	// Determine working directory
	workingDir := "."
	if len(args) > 0 {
		workingDir = args[0]
	}

	// Convert workspace type string to type
	var wsType cleanup.WorkspaceType
	switch statusWorkspaceType {
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
		return fmt.Errorf("unknown workspace type: %s", statusWorkspaceType)
	}

	// Get workspace state
	state, err := cleanup.GetWorkspaceState(workingDir, wsType)
	if err != nil {
		return fmt.Errorf("failed to get workspace state: %w", err)
	}

	// Run preflight check
	checker := cleanup.NewPreflightChecker(workingDir)
	checker.SetVerbose(workspaceStatusVerbose)
	checkResult, err := checker.Check()
	if err != nil {
		return fmt.Errorf("preflight check failed: %w", err)
	}

	// Output as JSON
	if workspaceStatusJSON {
		output := map[string]interface{}{
			"state":     state,
			"preflight": checkResult,
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	fmt.Printf("Workspace Status: %s\n", workingDir)
	fmt.Printf("Type: %s\n\n", state.Type)

	// Git status
	if state.GitStatus == "clean" {
		fmt.Printf("✅ Git: Clean\n")
	} else {
		fmt.Printf("⚠️  Git: %s\n", state.GitStatus)
		if state.DirtyFiles > 0 {
			fmt.Printf("   Modified files: %d\n", state.DirtyFiles)
		}
		if state.UntrackedFiles > 0 {
			fmt.Printf("   Untracked files: %d\n", state.UntrackedFiles)
		}
	}

	// Temp files
	if state.TempFileCount > 0 {
		fmt.Printf("⚠️  Temp files: %d (%.2f MB)\n", state.TempFileCount, float64(state.TempFileSize)/(1024*1024))
	} else {
		fmt.Printf("✅ Temp files: None\n")
	}

	// Preflight check
	fmt.Println()
	if checkResult.Passed {
		fmt.Printf("✅ Preflight: Passed\n")
	} else {
		fmt.Printf("❌ Preflight: Failed\n")
	}

	if checkResult.CanProceed {
		fmt.Printf("✅ Can proceed: Yes\n")
	} else {
		fmt.Printf("❌ Can proceed: No\n")
	}

	if workspaceStatusVerbose {
		if len(checkResult.Issues) > 0 {
			fmt.Println("\nIssues:")
			for _, issue := range checkResult.Issues {
				fmt.Printf("  - %s\n", issue)
			}
		}

		if len(checkResult.Warnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, warning := range checkResult.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}

		if len(checkResult.UncommittedFiles) > 0 {
			fmt.Println("\nUncommitted files:")
			for i, file := range checkResult.UncommittedFiles {
				if i >= 10 {
					fmt.Printf("  ... and %d more\n", len(checkResult.UncommittedFiles)-10)
					break
				}
				fmt.Printf("  %s\n", file)
			}
		}
	}

	if checkResult.RequiresBackup {
		fmt.Println("\n💡 Recommendation: Backup uncommitted changes before cleanup")
	}

	return nil
}
