package cmd

import (
	"fmt"

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
	// TODO: Implement workspace registration
	// rigName := args[0]
	// path := args[1]

	fmt.Println("Workspace registration:")
	fmt.Println("  This feature is not yet implemented.")
	fmt.Println("  Use 'gt crew add' or 'gt polecat identity add' for now.")

	return fmt.Errorf("workspace registration not yet implemented")
}
