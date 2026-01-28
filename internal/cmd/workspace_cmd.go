package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(NewWorkspaceCmd())
}

// NewWorkspaceCmd creates the workspace command.
func NewWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		GroupID: GroupWorkspace,
		Short:   "Workspace management commands",
		Long: `Manage workspace state, cleanup, and maintenance operations.

WORKSPACE TYPES:
  crew:     Persistent workspaces for humans (full git clones)
  polecat:  Ephemeral workspaces for agents (git worktrees)
  mayor:    Canonical read-only clone
  refinery: Merge queue worktree
  town:     Town-level workspace

Commands for workspace maintenance:
  gt workspace status [path]         Show workspace state and cleanliness
  gt workspace clean                 Clean temporary files
  gt workspace config                Manage workspace configuration`,
		RunE: requireSubcommand,
	}

	cmd.AddCommand(
		newWorkspaceCleanCmd(),
		newWorkspaceStatusCmd(),
		newWorkspaceConfigCmd(),
	)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewWorkspaceCmd())
}
