package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/style"
)

var (
	workspaceInitType   string
	workspaceInitBranch bool
)

func newWorkspaceInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <rig> <name>",
		Short: "Initialize a new workspace",
		Long: `Initialize a new workspace in the specified rig.

By default creates a crew workspace (persistent git clone). Use --type=polecat
for ephemeral worktree workspaces.

The workspace is created with:
  - Git repository (clone for crew, worktree for polecat)
  - .claude/settings.json for Claude Code integration
  - Beads context initialization
  - Git identity (Eric Friday)
  - Registration in town registry

Examples:
  gt workspace init duneagent dave                 # Create crew workspace
  gt workspace init duneagent toast --type polecat # Create polecat workspace
  gt workspace init duneagent emma --branch        # Create with feature branch`,
		Args: cobra.ExactArgs(2),
		RunE: runWorkspaceInit,
	}

	cmd.Flags().StringVar(&workspaceInitType, "type", "crew", "Workspace type: crew or polecat")
	cmd.Flags().BoolVar(&workspaceInitBranch, "branch", false, "Create a feature branch")

	return cmd
}

func runWorkspaceInit(cmd *cobra.Command, args []string) error {
	rigName := args[0]
	name := args[1]

	// Validate workspace type
	if workspaceInitType != "crew" && workspaceInitType != "polecat" {
		return fmt.Errorf("invalid workspace type: %s (must be 'crew' or 'polecat')", workspaceInitType)
	}

	fmt.Printf("Initializing %s workspace %s/%s...\n", workspaceInitType, rigName, name)

	// Delegate to appropriate command based on type
	if workspaceInitType == "crew" {
		// Set crew flags and run crew add
		crewRig = rigName
		crewBranch = workspaceInitBranch
		return runCrewAdd(cmd, []string{name})
	}

	// For polecat, create using polecat manager
	mgr, _, err := getPolecatManager(rigName)
	if err != nil {
		return err
	}

	p, err := mgr.Add(name)
	if err != nil {
		return fmt.Errorf("creating polecat workspace: %w", err)
	}

	fmt.Printf("%s Polecat workspace %s created.\n", style.SuccessPrefix, p.Name)
	fmt.Printf("  %s\n", style.Dim.Render(p.ClonePath))
	fmt.Printf("  Branch: %s\n", style.Dim.Render(p.Branch))

	return nil
}
