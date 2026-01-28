package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/crew"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/tmux"
)

var (
	workspaceListJSON bool
	workspaceListAll  bool
)

// WorkspaceInfo represents a workspace for listing output.
type WorkspaceInfo struct {
	Rig            string    `json:"rig"`
	Name           string    `json:"name"`
	Type           string    `json:"type"` // "crew" or "polecat"
	Path           string    `json:"path"`
	Branch         string    `json:"branch"`
	Status         string    `json:"status"`          // "active", "idle", "dirty", "stale"
	SessionRunning bool      `json:"session_running"`
	Dirty          bool      `json:"dirty"`
	CommitsBehind  int       `json:"commits_behind"`
	CommitsAhead   int       `json:"commits_ahead"`
	LastActivity   time.Time `json:"last_activity,omitempty"`
	Issue          string    `json:"issue,omitempty"`
}

func newWorkspaceListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [<rig>]",
		Short: "List all workspaces",
		Long: `List all workspaces in a rig or across all rigs.

Shows workspace name, type, status, current branch, uncommitted changes,
and last activity timestamp.

Status indicators:
  ● active   - Session running
  ○ idle     - Session stopped, workspace clean
  ⚠ dirty    - Uncommitted changes present
  ✗ stale    - Far behind main branch

Examples:
  gt workspace list                     # List in current rig
  gt workspace list duneagent           # List in specific rig
  gt workspace list --all               # List across all rigs
  gt workspace list --json              # JSON output`,
		Args: cobra.MaximumNArgs(1),
		RunE: runWorkspaceList,
	}

	cmd.Flags().BoolVar(&workspaceListAll, "all", false, "List workspaces in all rigs")
	cmd.Flags().BoolVar(&workspaceListJSON, "json", false, "Output as JSON")

	return cmd
}

func runWorkspaceList(cmd *cobra.Command, args []string) error {
	var rigs []string

	if workspaceListAll {
		// List all rigs
		allRigs, _, err := getAllRigs()
		if err != nil {
			return err
		}
		for _, r := range allRigs {
			rigs = append(rigs, r.Name)
		}
	} else if len(args) > 0 {
		rigs = []string{args[0]}
	} else {
		// Infer rig from current directory
		_, r, err := detectRig()
		if err != nil {
			return fmt.Errorf("could not detect rig from current directory: %w", err)
		}
		rigs = []string{r.Name}
	}

	// Collect workspaces from all rigs
	var allWorkspaces []WorkspaceInfo

	for _, rigName := range rigs {
		// Get crew workspaces
		crewWorkspaces, err := listCrewWorkspaces(rigName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list crew in %s: %v\n", rigName, err)
		} else {
			allWorkspaces = append(allWorkspaces, crewWorkspaces...)
		}

		// Get polecat workspaces
		polecatWorkspaces, err := listPolecatWorkspaces(rigName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list polecats in %s: %v\n", rigName, err)
		} else {
			allWorkspaces = append(allWorkspaces, polecatWorkspaces...)
		}
	}

	// Output
	if workspaceListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(allWorkspaces)
	}

	if len(allWorkspaces) == 0 {
		fmt.Println("No workspaces found.")
		return nil
	}

	fmt.Printf("%s\n\n", style.Bold.Render("Workspaces"))
	for _, ws := range allWorkspaces {
		// Status indicator
		statusIcon := style.Dim.Render("○")
		if ws.SessionRunning {
			statusIcon = style.Success.Render("●")
		} else if ws.Dirty {
			statusIcon = style.Warning.Render("⚠")
		}

		// Type badge
		typeBadge := style.Dim.Render("[crew]")
		if ws.Type == "polecat" {
			typeBadge = style.Info.Render("[cat]")
		}

		fmt.Printf("  %s %s/%s %s\n", statusIcon, ws.Rig, ws.Name, typeBadge)

		// Branch and status
		branchInfo := ws.Branch
		if ws.CommitsAhead > 0 || ws.CommitsBehind > 0 {
			branchInfo = fmt.Sprintf("%s (+%d/-%d)", ws.Branch, ws.CommitsAhead, ws.CommitsBehind)
		}
		fmt.Printf("    %s\n", style.Dim.Render(branchInfo))

		// Issue if assigned
		if ws.Issue != "" {
			fmt.Printf("    %s\n", style.Dim.Render(ws.Issue))
		}

		// Last activity
		if !ws.LastActivity.IsZero() {
			ago := formatRelativeTime(ws.LastActivity)
			fmt.Printf("    Last activity: %s\n", style.Dim.Render(ago))
		}
	}

	return nil
}

func listCrewWorkspaces(rigName string) ([]WorkspaceInfo, error) {
	_, r, err := getRig(rigName)
	if err != nil {
		return nil, err
	}

	crewDir := filepath.Join(r.Path, "crew")
	if _, err := os.Stat(crewDir); os.IsNotExist(err) {
		return nil, nil // No crew directory
	}

	t := tmux.NewTmux()
	crewGit := git.NewGit(r.Path)
	crewMgr := crew.NewManager(r, crewGit)

	// List all crew workers
	workers, err := crewMgr.List()
	if err != nil {
		return nil, err
	}

	var workspaces []WorkspaceInfo
	for _, worker := range workers {
		// Check session status
		sessionID := crewSessionName(r.Name, worker.Name)
		running, _ := t.HasSession(sessionID)

		// Get git status
		gitState, _ := getWorkspaceGitStatus(worker.ClonePath)

		ws := WorkspaceInfo{
			Rig:            rigName,
			Name:           worker.Name,
			Type:           "crew",
			Path:           worker.ClonePath,
			Branch:         gitState.Branch,
			SessionRunning: running,
			Dirty:          gitState.Dirty,
			CommitsBehind:  gitState.CommitsBehind,
			CommitsAhead:   gitState.CommitsAhead,
		}

		// Determine status
		if running {
			ws.Status = "active"
		} else if gitState.Dirty {
			ws.Status = "dirty"
		} else if gitState.CommitsBehind > 20 {
			ws.Status = "stale"
		} else {
			ws.Status = "idle"
		}

		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

func listPolecatWorkspaces(rigName string) ([]WorkspaceInfo, error) {
	mgr, r, err := getPolecatManager(rigName)
	if err != nil {
		return nil, err
	}

	polecats, err := mgr.List()
	if err != nil {
		return nil, err
	}

	t := tmux.NewTmux()
	polecatMgr := polecat.NewSessionManager(t, r)

	var workspaces []WorkspaceInfo
	for _, p := range polecats {
		running, _ := polecatMgr.IsRunning(p.Name)

		// Get git status
		gitState, _ := getWorkspaceGitStatus(p.ClonePath)

		ws := WorkspaceInfo{
			Rig:            rigName,
			Name:           p.Name,
			Type:           "polecat",
			Path:           p.ClonePath,
			Branch:         p.Branch,
			SessionRunning: running,
			Dirty:          gitState.Dirty,
			CommitsBehind:  gitState.CommitsBehind,
			CommitsAhead:   gitState.CommitsAhead,
			Issue:          p.Issue,
		}

		// Determine status based on polecat state
		switch p.State {
		case polecat.StateWorking:
			ws.Status = "active"
		case polecat.StateStuck:
			ws.Status = "stuck"
		case polecat.StateDone:
			ws.Status = "done"
		default:
			ws.Status = "idle"
		}

		// Get last activity
		if sessInfo, err := polecatMgr.Status(p.Name); err == nil && !sessInfo.LastActivity.IsZero() {
			ws.LastActivity = sessInfo.LastActivity
		}

		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

// GitStatusInfo represents git repository status.
type GitStatusInfo struct {
	Branch        string
	Dirty         bool
	CommitsBehind int
	CommitsAhead  int
}

func getWorkspaceGitStatus(wsPath string) (*GitStatusInfo, error) {
	status := &GitStatusInfo{}

	// Get current branch
	g := git.NewGit(wsPath)
	branch, err := g.CurrentBranch()
	if err != nil {
		return status, err
	}
	status.Branch = branch

	// Check if dirty
	dirty, err := g.IsDirty()
	if err == nil {
		status.Dirty = dirty
	}

	// Get commits ahead/behind
	// This is a simplified version - could be enhanced with actual git calls
	status.CommitsAhead = 0
	status.CommitsBehind = 0

	return status, nil
}
