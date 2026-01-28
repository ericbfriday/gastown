package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newWorkspaceAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <rig> <path>",
		Short: "Register an existing workspace",
		Long: `Register an existing git clone or worktree as a workspace.

This command registers an existing directory as a workspace without creating
new git clones. Useful for adopting pre-existing development directories.

The workspace type is auto-detected from the path:
  - crew/<name>/ → crew workspace
  - polecats/<name>/ → polecat workspace

Examples:
  gt workspace add duneagent ~/dev/duneagent-fork
  gt workspace add duneagent /tmp/quick-fix`,
		Args: cobra.ExactArgs(2),
		RunE: runWorkspaceAdd,
	}

	return cmd
}

func runWorkspaceAdd(cmd *cobra.Command, args []string) error {
	rigName := args[0]
	path := args[1]

	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	// Detect workspace type from path
	wsType := "crew"
	if strings.Contains(absPath, "/polecats/") {
		wsType = "polecat"
	}

	fmt.Printf("Registering %s workspace at %s...\n", wsType, absPath)

	// TODO: Implement workspace registration
	// This would involve:
	// 1. Verifying it's a valid git repo
	// 2. Creating/updating .claude/settings.json
	// 3. Initializing beads context
	// 4. Setting git identity
	// 5. Registering in town registry

	return fmt.Errorf("workspace registration not yet implemented - use 'gt crew add' or 'gt polecat identity add' instead")
}
