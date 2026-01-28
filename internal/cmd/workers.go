package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/monitoring"
	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/tmux"
)

// Workers command flags
var (
	workersJSON   bool
	workersVerbose bool
	workersRig    string
)

var workersCmd = &cobra.Command{
	Use:     "workers",
	GroupID: GroupAgents,
	Short:   "Monitor and report on worker status (crew/polecats)",
	RunE:    requireSubcommand,
	Long: `Monitor and report on worker status across all rigs.

Workers include both crew members (persistent workspaces) and polecats
(ephemeral workers). This command provides visibility into what agents
are doing, resource utilization, and system health.

Commands:
  gt workers list [rig]     List all workers with basic status
  gt workers status <rig>/<name>  Detailed worker status
  gt workers active         Show only active workers
  gt workers health         Overall system health check`,
}

var workersListCmd = &cobra.Command{
	Use:   "list [rig]",
	Short: "List all workers with basic status",
	Long: `List all workers (crew and polecats) with their current status.

If a rig is specified, lists only workers in that rig.
Otherwise, lists workers across all rigs.

Output includes:
  - Worker name and type (crew/polecat)
  - Current state (idle/working/stalled/crashed)
  - Current assignment (issue ID)
  - Last activity timestamp
  - Session status (running/stopped)

Examples:
  gt workers list                  # List all workers
  gt workers list duneagent        # List workers in duneagent rig
  gt workers list --json           # JSON output`,
	RunE: runWorkersList,
}

var workersStatusCmd = &cobra.Command{
	Use:   "status <rig>/<name>",
	Short: "Show detailed status for a specific worker",
	Long: `Show detailed status for a crew or polecat worker.

Displays comprehensive information including:
  - Worker type, rig, and name
  - Current state and activity status
  - Current assignment (issue ID and description)
  - Session information (running/stopped, session ID)
  - Git status (branch, uncommitted changes, commits ahead)
  - Last activity timestamp and duration
  - Resource usage (if available)

Examples:
  gt workers status duneagent/ericfriday
  gt workers status duneagent/rust
  gt workers status duneagent/ericfriday --json`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkersStatus,
}

var workersActiveCmd = &cobra.Command{
	Use:   "active",
	Short: "Show only active workers",
	Long: `Show only workers that are currently active (have running sessions).

Filters out stopped/idle workers to focus on current activity.
Useful for checking what's happening right now.

Examples:
  gt workers active                # Show active workers
  gt workers active --rig duneagent  # Active workers in specific rig
  gt workers active --json         # JSON output`,
	RunE: runWorkersActive,
}

var workersHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Overall system health check",
	Long: `Aggregate health status across all workers.

Reports:
  - Total workers, active count, utilization percentage
  - Stalled/crashed workers needing attention
  - Resource bottlenecks (if available)
  - Recommendations for action

Useful for getting a quick overview of system health.

Examples:
  gt workers health                # Health check
  gt workers health --json         # JSON output`,
	RunE: runWorkersHealth,
}

func init() {
	// List flags
	workersListCmd.Flags().BoolVar(&workersJSON, "json", false, "Output as JSON")

	// Status flags
	workersStatusCmd.Flags().BoolVar(&workersJSON, "json", false, "Output as JSON")

	// Active flags
	workersActiveCmd.Flags().BoolVar(&workersJSON, "json", false, "Output as JSON")
	workersActiveCmd.Flags().StringVar(&workersRig, "rig", "", "Filter by rig name")

	// Health flags
	workersHealthCmd.Flags().BoolVar(&workersJSON, "json", false, "Output as JSON")

	// Add subcommands
	workersCmd.AddCommand(workersListCmd)
	workersCmd.AddCommand(workersStatusCmd)
	workersCmd.AddCommand(workersActiveCmd)
	workersCmd.AddCommand(workersHealthCmd)

	rootCmd.AddCommand(workersCmd)
}

// WorkerInfo represents worker information for display.
type WorkerInfo struct {
	Rig            string                  `json:"rig"`
	Name           string                  `json:"name"`
	Type           string                  `json:"type"` // "crew" or "polecat"
	State          string                  `json:"state"`
	Status         monitoring.AgentStatus  `json:"status,omitempty"`
	Issue          string                  `json:"issue,omitempty"`
	Branch         string                  `json:"branch,omitempty"`
	SessionRunning bool                    `json:"session_running"`
	SessionID      string                  `json:"session_id,omitempty"`
	LastActivity   time.Time               `json:"last_activity,omitempty"`
	GitStatus      *WorkerGitStatus        `json:"git_status,omitempty"`
}

// WorkerGitStatus represents git status for a worker.
type WorkerGitStatus struct {
	Branch           string `json:"branch"`
	UncommittedFiles int    `json:"uncommitted_files"`
	CommitsAhead     int    `json:"commits_ahead"`
	IsDirty          bool   `json:"is_dirty"`
}

// WorkerHealth represents aggregate health status.
type WorkerHealth struct {
	TotalWorkers   int      `json:"total_workers"`
	ActiveWorkers  int      `json:"active_workers"`
	IdleWorkers    int      `json:"idle_workers"`
	StalledWorkers int      `json:"stalled_workers"`
	ErrorWorkers   int      `json:"error_workers"`
	Utilization    float64  `json:"utilization_percent"`
	Problems       []string `json:"problems,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}

func runWorkersList(cmd *cobra.Command, args []string) error {
	var rigs []*rig.Rig

	if len(args) > 0 {
		// Specific rig
		_, r, err := getRig(args[0])
		if err != nil {
			return err
		}
		rigs = []*rig.Rig{r}
	} else {
		// All rigs
		allRigs, _, err := getAllRigs()
		if err != nil {
			return err
		}
		rigs = allRigs
	}

	// Collect workers from all rigs
	workers, err := collectWorkers(rigs)
	if err != nil {
		return err
	}

	if len(workers) == 0 {
		fmt.Println("No workers found.")
		return nil
	}

	// Output
	if workersJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(workers)
	}

	displayWorkerList(workers, "Workers")
	return nil
}

func runWorkersStatus(cmd *cobra.Command, args []string) error {
	rigName, workerName, err := parseAddress(args[0])
	if err != nil {
		return err
	}

	_, r, err := getRig(rigName)
	if err != nil {
		return err
	}

	// Try to find worker (could be crew or polecat)
	worker, err := getWorkerInfo(r, workerName)
	if err != nil {
		return err
	}

	// Get detailed git status
	gitStatus, err := getWorkerGitStatus(worker)
	if err == nil {
		worker.GitStatus = gitStatus
	}

	// Output
	if workersJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(worker)
	}

	displayWorkerDetail(worker)
	return nil
}

func runWorkersActive(cmd *cobra.Command, args []string) error {
	var rigs []*rig.Rig

	if workersRig != "" {
		// Specific rig
		_, r, err := getRig(workersRig)
		if err != nil {
			return err
		}
		rigs = []*rig.Rig{r}
	} else {
		// All rigs
		allRigs, _, err := getAllRigs()
		if err != nil {
			return err
		}
		rigs = allRigs
	}

	// Collect workers
	allWorkers, err := collectWorkers(rigs)
	if err != nil {
		return err
	}

	// Filter to active only
	var activeWorkers []*WorkerInfo
	for _, w := range allWorkers {
		if w.SessionRunning {
			activeWorkers = append(activeWorkers, w)
		}
	}

	if len(activeWorkers) == 0 {
		fmt.Println("No active workers found.")
		return nil
	}

	// Output
	if workersJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(activeWorkers)
	}

	displayWorkerList(activeWorkers, "Active Workers")
	return nil
}

func runWorkersHealth(cmd *cobra.Command, args []string) error {
	// Get all rigs
	allRigs, _, err := getAllRigs()
	if err != nil {
		return err
	}

	// Collect workers
	workers, err := collectWorkers(allRigs)
	if err != nil {
		return err
	}

	// Calculate health metrics
	health := calculateHealth(workers)

	// Output
	if workersJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(health)
	}

	displayHealth(health)
	return nil
}

// collectWorkers gathers worker info from all specified rigs.
func collectWorkers(rigs []*rig.Rig) ([]*WorkerInfo, error) {
	var workers []*WorkerInfo
	t := tmux.NewTmux()

	for _, r := range rigs {
		// Collect crew workers
		crewPath := filepath.Join(r.Path, "crew")
		if entries, err := os.ReadDir(crewPath); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
					continue
				}

				worker, err := getCrewWorkerInfo(r, entry.Name(), t)
				if err != nil {
					continue
				}
				workers = append(workers, worker)
			}
		}

		// Collect polecat workers
		polecatGit := git.NewGit(r.Path)
		mgr := polecat.NewManager(r, polecatGit, t)
		polecats, err := mgr.List()
		if err != nil {
			continue
		}

		for _, p := range polecats {
			worker, err := getPolecatWorkerInfo(r, &p, t)
			if err != nil {
				continue
			}
			workers = append(workers, worker)
		}
	}

	return workers, nil
}

// getWorkerInfo finds a specific worker by name.
func getWorkerInfo(r *rig.Rig, name string) (*WorkerInfo, error) {
	t := tmux.NewTmux()

	// Try crew first
	crewPath := filepath.Join(r.Path, "crew", name)
	if info, err := os.Stat(crewPath); err == nil && info.IsDir() {
		return getCrewWorkerInfo(r, name, t)
	}

	// Try polecat
	polecatGit := git.NewGit(r.Path)
	mgr := polecat.NewManager(r, polecatGit, t)
	p, err := mgr.Get(name)
	if err == nil {
		return getPolecatWorkerInfo(r, p, t)
	}

	return nil, fmt.Errorf("worker '%s' not found in rig '%s'", name, r.Name)
}

// getCrewWorkerInfo gathers info for a crew worker.
func getCrewWorkerInfo(r *rig.Rig, name string, t *tmux.Tmux) (*WorkerInfo, error) {
	worker := &WorkerInfo{
		Rig:  r.Name,
		Name: name,
		Type: "crew",
		State: "idle",
	}

	// Check for running session
	sessionID := fmt.Sprintf("gt-%s-crew-%s", r.Name, name)
	running, _ := t.HasSession(sessionID)
	worker.SessionRunning = running
	if running {
		worker.SessionID = sessionID
		worker.State = "working"

		// Get last activity
		if sessInfo, err := t.GetSessionInfo(sessionID); err == nil {
			worker.LastActivity = sessInfo.LastActivity
		}
	}

	// Get git branch
	crewPath := filepath.Join(r.Path, "crew", name)
	g := git.NewGit(crewPath)
	if branch, err := g.CurrentBranch(); err == nil {
		worker.Branch = branch
	}

	return worker, nil
}

// getPolecatWorkerInfo gathers info for a polecat worker.
func getPolecatWorkerInfo(r *rig.Rig, p *polecat.Polecat, t *tmux.Tmux) (*WorkerInfo, error) {
	worker := &WorkerInfo{
		Rig:    r.Name,
		Name:   p.Name,
		Type:   "polecat",
		State:  string(p.State),
		Issue:  p.Issue,
		Branch: p.Branch,
	}

	// Check for running session
	sessionID := fmt.Sprintf("gt-%s-%s", r.Name, p.Name)
	running, _ := t.HasSession(sessionID)
	worker.SessionRunning = running
	if running {
		worker.SessionID = sessionID

		// Get last activity
		if sessInfo, err := t.GetSessionInfo(sessionID); err == nil {
			worker.LastActivity = sessInfo.LastActivity
		}
	}

	return worker, nil
}

// getWorkerGitStatus gets detailed git status for a worker.
func getWorkerGitStatus(worker *WorkerInfo) (*WorkerGitStatus, error) {
	var workPath string
	if worker.Type == "crew" {
		workPath = filepath.Join(worker.Rig, "crew", worker.Name)
	} else {
		workPath = filepath.Join(worker.Rig, "polecats", worker.Name)
	}

	// Make path absolute from town root
	townRoot, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	workPath = filepath.Join(townRoot, workPath)

	g := git.NewGit(workPath)
	status := &WorkerGitStatus{
		Branch: worker.Branch,
	}

	// Check for uncommitted files
	if dirty, err := g.IsDirty(); err == nil && dirty {
		status.IsDirty = true
		// Count files
		if files, err := g.Status(); err == nil {
			status.UncommittedFiles = len(files)
		}
	}

	// Check commits ahead
	if ahead, err := g.CommitsAhead("origin/main"); err == nil {
		status.CommitsAhead = ahead
	}

	return status, nil
}

// calculateHealth computes aggregate health metrics.
func calculateHealth(workers []*WorkerInfo) *WorkerHealth {
	health := &WorkerHealth{
		TotalWorkers: len(workers),
	}

	for _, w := range workers {
		if w.SessionRunning {
			health.ActiveWorkers++
		}

		switch w.State {
		case "idle", "available":
			health.IdleWorkers++
		case "stuck", "stalled":
			health.StalledWorkers++
			health.Problems = append(health.Problems,
				fmt.Sprintf("%s/%s is stalled", w.Rig, w.Name))
		case "error", "crashed":
			health.ErrorWorkers++
			health.Problems = append(health.Problems,
				fmt.Sprintf("%s/%s has crashed", w.Rig, w.Name))
		}
	}

	// Calculate utilization
	if health.TotalWorkers > 0 {
		health.Utilization = float64(health.ActiveWorkers) / float64(health.TotalWorkers) * 100
	}

	// Generate recommendations
	if health.StalledWorkers > 0 {
		health.Recommendations = append(health.Recommendations,
			fmt.Sprintf("Investigate %d stalled worker(s)", health.StalledWorkers))
	}
	if health.ErrorWorkers > 0 {
		health.Recommendations = append(health.Recommendations,
			fmt.Sprintf("Restart %d crashed worker(s)", health.ErrorWorkers))
	}
	if health.Utilization < 30 && health.TotalWorkers > 2 {
		health.Recommendations = append(health.Recommendations,
			"Low utilization - consider assigning more work")
	}

	return health
}

// displayWorkerList shows a table of workers.
func displayWorkerList(workers []*WorkerInfo, title string) {
	fmt.Printf("%s\n\n", style.Bold.Render(title))

	for _, w := range workers {
		// Session indicator
		sessionStatus := style.Dim.Render("○")
		if w.SessionRunning {
			sessionStatus = style.Success.Render("●")
		}

		// State color
		stateStr := w.State
		switch w.State {
		case "working":
			stateStr = style.Info.Render(stateStr)
		case "stuck", "stalled":
			stateStr = style.Warning.Render(stateStr)
		case "error", "crashed":
			stateStr = style.Error.Render(stateStr)
		default:
			stateStr = style.Dim.Render(stateStr)
		}

		// Type badge
		typeBadge := "crew"
		if w.Type == "polecat" {
			typeBadge = "polecat"
		}

		fmt.Printf("  %s %s/%s [%s]  %s\n",
			sessionStatus, w.Rig, w.Name, typeBadge, stateStr)

		// Show issue if present
		if w.Issue != "" {
			fmt.Printf("    %s\n", style.Dim.Render(w.Issue))
		}

		// Show last activity if available
		if !w.LastActivity.IsZero() {
			ago := formatActivityTime(w.LastActivity)
			fmt.Printf("    Last activity: %s\n", style.Dim.Render(ago))
		}
	}

	fmt.Printf("\nTotal: %d workers\n", len(workers))
}

// displayWorkerDetail shows detailed worker information.
func displayWorkerDetail(worker *WorkerInfo) {
	fmt.Printf("%s\n\n", style.Bold.Render(fmt.Sprintf("Worker: %s/%s", worker.Rig, worker.Name)))

	fmt.Printf("  Type:          %s\n", worker.Type)

	// State with color
	stateStr := worker.State
	switch worker.State {
	case "working":
		stateStr = style.Info.Render(stateStr)
	case "stuck", "stalled":
		stateStr = style.Warning.Render(stateStr)
	case "error", "crashed":
		stateStr = style.Error.Render(stateStr)
	default:
		stateStr = style.Dim.Render(stateStr)
	}
	fmt.Printf("  State:         %s\n", stateStr)

	// Issue
	if worker.Issue != "" {
		fmt.Printf("  Issue:         %s\n", worker.Issue)
	}

	// Branch
	if worker.Branch != "" {
		fmt.Printf("  Branch:        %s\n", style.Dim.Render(worker.Branch))
	}

	// Session info
	fmt.Println()
	fmt.Printf("%s\n", style.Bold.Render("Session"))
	if worker.SessionRunning {
		fmt.Printf("  Status:        %s\n", style.Success.Render("running"))
		if worker.SessionID != "" {
			fmt.Printf("  Session ID:    %s\n", style.Dim.Render(worker.SessionID))
		}
		if !worker.LastActivity.IsZero() {
			ago := formatActivityTime(worker.LastActivity)
			fmt.Printf("  Last Activity: %s (%s)\n",
				worker.LastActivity.Format("15:04:05"),
				style.Dim.Render(ago))
		}
	} else {
		fmt.Printf("  Status:        %s\n", style.Dim.Render("not running"))
	}

	// Git status
	if worker.GitStatus != nil {
		fmt.Println()
		fmt.Printf("%s\n", style.Bold.Render("Git Status"))

		if worker.GitStatus.IsDirty {
			fmt.Printf("  Working Tree:  %s\n", style.Warning.Render("dirty"))
			if worker.GitStatus.UncommittedFiles > 0 {
				fmt.Printf("  Uncommitted:   %s\n",
					style.Warning.Render(fmt.Sprintf("%d files", worker.GitStatus.UncommittedFiles)))
			}
		} else {
			fmt.Printf("  Working Tree:  %s\n", style.Success.Render("clean"))
		}

		if worker.GitStatus.CommitsAhead > 0 {
			fmt.Printf("  Commits Ahead: %s\n",
				style.Info.Render(fmt.Sprintf("%d", worker.GitStatus.CommitsAhead)))
		}
	}
}

// displayHealth shows health status.
func displayHealth(health *WorkerHealth) {
	fmt.Printf("%s\n\n", style.Bold.Render("System Health"))

	fmt.Printf("  Total Workers:    %d\n", health.TotalWorkers)
	fmt.Printf("  Active:           %s\n", style.Success.Render(fmt.Sprintf("%d", health.ActiveWorkers)))
	fmt.Printf("  Idle:             %s\n", style.Dim.Render(fmt.Sprintf("%d", health.IdleWorkers)))

	if health.StalledWorkers > 0 {
		fmt.Printf("  Stalled:          %s\n", style.Warning.Render(fmt.Sprintf("%d", health.StalledWorkers)))
	}

	if health.ErrorWorkers > 0 {
		fmt.Printf("  Errors:           %s\n", style.Error.Render(fmt.Sprintf("%d", health.ErrorWorkers)))
	}

	fmt.Printf("  Utilization:      %.1f%%\n", health.Utilization)

	// Problems
	if len(health.Problems) > 0 {
		fmt.Println()
		fmt.Printf("%s\n", style.Bold.Render("Problems"))
		for _, problem := range health.Problems {
			fmt.Printf("  %s %s\n", style.Warning.Render("⚠"), problem)
		}
	}

	// Recommendations
	if len(health.Recommendations) > 0 {
		fmt.Println()
		fmt.Printf("%s\n", style.Bold.Render("Recommendations"))
		for _, rec := range health.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
	}

	// Overall verdict
	fmt.Println()
	if health.ErrorWorkers > 0 || health.StalledWorkers > 0 {
		fmt.Printf("  Overall:          %s\n", style.Error.Render("ATTENTION NEEDED"))
	} else if health.ActiveWorkers == 0 {
		fmt.Printf("  Overall:          %s\n", style.Warning.Render("IDLE"))
	} else {
		fmt.Printf("  Overall:          %s\n", style.Success.Render("HEALTHY"))
	}
}
